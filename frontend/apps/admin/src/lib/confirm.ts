import { shallowRef } from 'vue'

export interface ConfirmOptions {
  title: string
  message?: string
  /** Butir penjelasan tambahan (mis. apa yang akan terjadi). */
  details?: string[]
  confirmText?: string
  cancelText?: string
  tone?: 'primary' | 'danger' | 'success'
}

interface PendingConfirm extends ConfirmOptions {
  resolve: (ok: boolean) => void
}

export const pendingConfirm = shallowRef<PendingConfirm | null>(null)

export function confirmDialog(opts: ConfirmOptions): Promise<boolean> {
  pendingConfirm.value?.resolve(false)
  return new Promise((resolve) => {
    pendingConfirm.value = { ...opts, resolve }
  })
}

export function settleConfirm(ok: boolean) {
  const p = pendingConfirm.value
  pendingConfirm.value = null
  p?.resolve(ok)
}
