import { reactive } from 'vue'
import { api, type User } from '@undangan/shared'

export const auth = reactive<{ user: User | null; checked: boolean }>({ user: null, checked: false })

export async function fetchMe(): Promise<User | null> {
  try {
    const res = await api.get<{ user: User }>('/auth/me')
    auth.user = res.user
  } catch {
    auth.user = null
  } finally {
    auth.checked = true
  }
  return auth.user
}

export class RoleError extends Error {}

export async function login(email: string, password: string) {
  const res = await api.post<{ user: User }>('/auth/login', { email, password, portal: 'admin' })
  if (res.user.role !== 'super_admin') {
    await logout()
    throw new RoleError('Akun ini tidak memiliki akses ke Admin Console.')
  }
  auth.user = res.user
  auth.checked = true
  return res.user
}

export async function logout() {
  try {
    await api.post('/auth/logout')
  } catch {
    /* sesi mungkin sudah habis */
  }
  auth.user = null
}
