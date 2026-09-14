import { reactive, readonly } from 'vue'
import { errorMessage } from '../utils/misc'

export type ToastKind = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  kind: ToastKind
  message: string
}

const state = reactive<{ items: ToastItem[] }>({ items: [] })
let seq = 0

function push(kind: ToastKind, message: string, timeout = kind === 'error' ? 6000 : 3500) {
  const id = ++seq
  state.items.push({ id, kind, message })
  if (state.items.length > 5) state.items.shift()
  if (timeout > 0) setTimeout(() => dismiss(id), timeout)
  return id
}

export function dismiss(id: number) {
  const i = state.items.findIndex((t) => t.id === id)
  if (i >= 0) state.items.splice(i, 1)
}

export const toast = {
  success: (message: string) => push('success', message),
  /** Terima string atau error (ApiError dsb.) */
  error: (message: unknown) => push('error', typeof message === 'string' ? message : errorMessage(message)),
  info: (message: string) => push('info', message),
  dismiss,
}

export const toastState = readonly(state)

export function useToast() {
  return toast
}
