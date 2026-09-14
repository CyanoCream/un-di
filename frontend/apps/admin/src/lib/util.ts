import { ApiError } from '@undangan/shared'
import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

export function errMsg(e: unknown, fallback = 'Gagal terhubung ke server.') {
  if (e instanceof ApiError) return e.message
  if (e instanceof Error && e.message && !(e instanceof TypeError)) return e.message
  return fallback
}

export function fieldErrors(e: unknown): Record<string, string> {
  return e instanceof ApiError ? e.fields : {}
}

/** Selisih hari (dibulatkan ke atas) dari sekarang ke tanggal ISO. Negatif = sudah lewat. */
export function daysUntil(iso: string | null | undefined): number | null {
  if (!iso) return null
  return Math.ceil((new Date(iso).getTime() - Date.now()) / 86_400_000)
}

export function relativeTime(iso: string | null | undefined) {
  if (!iso) return '-'
  const diff = (new Date(iso).getTime() - Date.now()) / 1000
  const abs = Math.abs(diff)
  const rtf = new Intl.RelativeTimeFormat('id-ID', { numeric: 'auto' })
  if (abs < 60) return 'baru saja'
  if (abs < 3600) return rtf.format(Math.round(diff / 60), 'minute')
  if (abs < 86400) return rtf.format(Math.round(diff / 3600), 'hour')
  if (abs < 86400 * 30) return rtf.format(Math.round(diff / 86400), 'day')
  return rtf.format(Math.round(diff / (86400 * 30)), 'month')
}

export function formatNumber(n: number | null | undefined) {
  return new Intl.NumberFormat('id-ID').format(n ?? 0)
}

export function formatDuration(sec: number | null | undefined) {
  const s = Math.max(0, Math.round(sec ?? 0))
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
}

export function initials(name: string | null | undefined) {
  const parts = (name ?? '').trim().split(/\s+/).filter(Boolean)
  return ((parts[0]?.[0] ?? '?') + (parts[1]?.[0] ?? '')).toUpperCase()
}

export function generatePassword(len = 12) {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789'
  const buf = new Uint32Array(len)
  crypto.getRandomValues(buf)
  return Array.from(buf, (n) => chars[n % chars.length]).join('')
}

export async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    ta.remove()
    return ok
  }
}

export function useDebounced<T>(source: Ref<T>, delay = 300): Ref<T> {
  const out = ref(source.value) as Ref<T>
  let t: number | undefined
  watch(source, (v) => {
    window.clearTimeout(t)
    t = window.setTimeout(() => (out.value = v), delay)
  })
  onBeforeUnmount(() => window.clearTimeout(t))
  return out
}

/** Normalisasi respons list yang bisa berupa array atau {items}. */
export function itemsOf<T>(res: T[] | { items: T[] } | null | undefined): T[] {
  if (!res) return []
  return Array.isArray(res) ? res : (res.items ?? [])
}
