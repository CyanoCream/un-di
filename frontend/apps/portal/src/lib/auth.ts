import { api, type User } from '@undangan/shared'
import { reactive } from 'vue'

export const auth = reactive<{ user: User | null; ready: boolean }>({ user: null, ready: false })

let sessionPromise: Promise<void> | null = null

/** Ambil sesi dari /auth/me sekali saja (dipakai router guard). */
export function ensureSession() {
  if (!sessionPromise) {
    sessionPromise = api
      .get<{ user: User }>('/auth/me')
      .then((r) => {
        auth.user = r.user
      })
      .catch(() => {
        auth.user = null
      })
      .finally(() => {
        auth.ready = true
      })
  }
  return sessionPromise
}

export async function login(email: string, password: string) {
  const r = await api.post<{ user: User }>('/auth/login', { email, password, portal: 'customer' })
  auth.user = r.user
  sessionPromise = Promise.resolve()
  return r.user
}

export async function register(input: { name: string; email: string; phone: string; password: string }) {
  const r = await api.post<{ user: User }>('/auth/register', input)
  auth.user = r.user
  sessionPromise = Promise.resolve()
  return r.user
}

export async function logout() {
  try {
    await api.post('/auth/logout')
  } catch {
    /* sesi mungkin sudah habis */
  }
  clearSession()
}

export function clearSession() {
  auth.user = null
  sessionPromise = Promise.resolve()
}

export function firstName(u: User | null) {
  return (u?.name ?? '').trim().split(/\s+/)[0] ?? ''
}
