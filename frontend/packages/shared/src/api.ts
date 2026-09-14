import type { ApiErrorBody } from './types'

export const API_BASE = '/api/v1'

export class ApiError extends Error {
  status: number
  code: string
  fields: Record<string, string>

  constructor(status: number, body: ApiErrorBody | null) {
    super(body?.error.message ?? `Terjadi kesalahan (${status})`)
    this.status = status
    this.code = body?.error.code ?? 'unknown'
    this.fields = body?.error.fields ?? {}
  }
}

type Query = Record<string, string | number | boolean | undefined | null>

function buildUrl(path: string, query?: Query) {
  const url = new URL(API_BASE + path, window.location.origin)
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== '') url.searchParams.set(k, String(v))
    }
  }
  return url.pathname + url.search
}

/** Dipanggil saat API membalas 401 (sesi habis). App memasang handler redirect ke login. */
let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

// Endpoint auth yang tidak boleh memicu refresh (menghindari loop).
const NO_REFRESH = ['/auth/login', '/auth/register', '/auth/refresh', '/auth/logout']

// Satu refresh untuk banyak request paralel yang kena 401 bersamaan.
let refreshing: Promise<boolean> | null = null
function refreshSession(): Promise<boolean> {
  refreshing ??= fetch(API_BASE + '/auth/refresh', {
    method: 'POST',
    headers: { 'X-Requested-With': 'fetch' },
    credentials: 'same-origin',
  })
    .then((r) => r.ok)
    .catch(() => false)
    .finally(() => {
      refreshing = null
    })
  return refreshing
}

async function request<T>(method: string, path: string, opts: { query?: Query; body?: unknown } = {}, retried = false): Promise<T> {
  const headers: Record<string, string> = { 'X-Requested-With': 'fetch' }
  let body: BodyInit | undefined
  if (opts.body instanceof FormData) {
    body = opts.body
  } else if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(opts.body)
  }

  const res = await fetch(buildUrl(path, opts.query), { method, headers, body, credentials: 'same-origin' })

  if (res.status === 401 && !NO_REFRESH.includes(path)) {
    // Access token JWT berumur pendek: coba perbarui sekali lewat refresh token (cookie), lalu ulangi request.
    if (!retried && (await refreshSession())) return request<T>(method, path, opts, true)
    if (onUnauthorized && path !== '/auth/me') onUnauthorized()
  }
  if (res.status === 204) return undefined as T

  const isJson = res.headers.get('Content-Type')?.includes('application/json')
  const data = isJson ? await res.json() : null
  if (!res.ok) throw new ApiError(res.status, data)
  return data as T
}

export const api = {
  get: <T>(path: string, query?: Query) => request<T>('GET', path, { query }),
  post: <T>(path: string, body?: unknown, query?: Query) => request<T>('POST', path, { body, query }),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, { body }),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, { body }),
  del: <T = void>(path: string) => request<T>('DELETE', path),
  /** Upload file; `fields` ikut sebagai form field tambahan. */
  upload: <T>(path: string, file: Blob, fields: Record<string, string> = {}, query?: Query, filename = 'file') => {
    const fd = new FormData()
    fd.append('file', file, (file as File).name ?? filename)
    for (const [k, v] of Object.entries(fields)) fd.append(k, v)
    return request<T>('POST', path, { body: fd, query })
  },
  url: buildUrl,
}

export function formatRupiah(n: number) {
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(n)
}

export function formatDate(iso: string | null | undefined, withTime = false) {
  if (!iso) return '-'
  const d = new Date(iso)
  return new Intl.DateTimeFormat('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    ...(withTime ? { hour: '2-digit', minute: '2-digit' } : {}),
  }).format(d)
}
