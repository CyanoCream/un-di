// Klien API stasiun check-in (penerima tamu).
//
// Sengaja TIDAK memakai `api` dari api.ts: stasiun dibuka oleh penerima tamu tanpa akun, jadi 401 di sini
// tidak boleh memicu handler "sesi habis → redirect ke login" milik aplikasi. Autentikasi:
//  1. Token stasiun (hasil PIN) di sessionStorage → dikirim sebagai `Authorization: Bearer <token>`.
//  2. Tanpa token → cookie sesi biasa (pemilik/admin yang sedang login), dengan satu kali percobaan refresh.

import { API_BASE, ApiError } from './api'
import type { ApiErrorBody, CheckinGuest, CheckinLog, CheckinResult, CheckinSession, CheckinSummary } from './types'

const STATION_NAME_KEY = 'checkin:station'

export function checkinTokenKey(invitationId: string) {
  return `checkin:${invitationId}`
}

function storageGet(store: Storage | undefined, key: string) {
  try {
    return store?.getItem(key) ?? null
  } catch {
    return null
  }
}

function storageSet(store: Storage | undefined, key: string, value: string | null) {
  try {
    if (value === null) store?.removeItem(key)
    else store?.setItem(key, value)
  } catch {
    /* mode privat / storage penuh — abaikan */
  }
}

export function rememberedStationName() {
  return storageGet(globalThis.localStorage, STATION_NAME_KEY) ?? ''
}

export function rememberStationName(name: string) {
  storageSet(globalThis.localStorage, STATION_NAME_KEY, name.trim() || null)
}

export function createCheckinClient(invitationId: string) {
  const base = `/checkin/${encodeURIComponent(invitationId)}`
  const key = checkinTokenKey(invitationId)

  const getToken = () => storageGet(globalThis.sessionStorage, key)
  const setToken = (t: string | null) => storageSet(globalThis.sessionStorage, key, t)

  async function refreshCookieSession() {
    try {
      const r = await fetch(API_BASE + '/auth/refresh', {
        method: 'POST',
        headers: { 'X-Requested-With': 'fetch' },
        credentials: 'same-origin',
      })
      return r.ok
    } catch {
      return false
    }
  }

  async function request<T>(method: string, path: string, body?: unknown, query?: Record<string, string>, retried = false): Promise<T> {
    const headers: Record<string, string> = { 'X-Requested-With': 'fetch' }
    const token = getToken()
    if (token) headers.Authorization = `Bearer ${token}`
    if (body !== undefined) headers['Content-Type'] = 'application/json'

    let url = API_BASE + base + path
    if (query) {
      const qs = new URLSearchParams(Object.entries(query).filter(([, v]) => v !== '')).toString()
      if (qs) url += `?${qs}`
    }

    const res = await fetch(url, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'same-origin',
    })

    if (res.status === 401) {
      if (token) {
        // Token stasiun kedaluwarsa / PIN diganti → wajib masukkan PIN lagi.
        setToken(null)
      } else if (!retried && path !== '/session' && (await refreshCookieSession())) {
        return request<T>(method, path, body, query, true)
      }
    }
    if (res.status === 204) return undefined as T

    const isJson = res.headers.get('Content-Type')?.includes('application/json')
    const data = isJson ? await res.json().catch(() => null) : null
    if (!res.ok) throw new ApiError(res.status, data as ApiErrorBody | null)
    return data as T
  }

  return {
    invitationId,
    hasToken: () => !!getToken(),
    logout: () => setToken(null),

    /** Tukar PIN dengan token stasiun (disimpan di sessionStorage). */
    async login(pin: string, stationName: string) {
      setToken(null)
      const s = await request<CheckinSession>('POST', '/session', { pin, station_name: stationName.trim() || undefined })
      setToken(s.token)
      return s
    },

    summary: () => request<CheckinSummary>('GET', '/summary'),
    searchGuests: (q: string) => request<{ items: CheckinGuest[] | null }>('GET', '/guests', undefined, { q }),
    /** `code` boleh berupa passcode 6 karakter atau isi QR mentah (URL dengan ?c=KODE). */
    scan: (code: string, pax?: number) => request<CheckinResult>('POST', '/scan', pax ? { code, pax } : { code }),
    recent: () => request<{ items: CheckinLog[] | null }>('GET', '/recent'),
  }
}

export type CheckinClient = ReturnType<typeof createCheckinClient>

/** Butuh autentikasi ulang (PIN) bila 401/403. */
export function isCheckinAuthError(e: unknown) {
  return e instanceof ApiError && (e.status === 401 || e.status === 403)
}
