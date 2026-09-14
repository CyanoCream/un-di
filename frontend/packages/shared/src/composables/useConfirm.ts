import { reactive } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

interface ConfirmState {
  open: boolean
  options: Required<ConfirmOptions>
  resolve: ((v: boolean) => void) | null
}

export const confirmState = reactive<ConfirmState>({
  open: false,
  options: { title: 'Konfirmasi', message: '', confirmText: 'Ya, lanjutkan', cancelText: 'Batal', danger: false },
  resolve: null,
})

/** `await confirm({...})` → true bila pengguna menyetujui. Butuh <UiConfirm /> terpasang sekali di App. */
export function confirm(options: ConfirmOptions): Promise<boolean> {
  // Tutup dialog sebelumnya (jika ada) sebagai "batal".
  confirmState.resolve?.(false)
  confirmState.options = {
    title: options.title ?? 'Konfirmasi',
    message: options.message,
    confirmText: options.confirmText ?? (options.danger ? 'Hapus' : 'Ya, lanjutkan'),
    cancelText: options.cancelText ?? 'Batal',
    danger: options.danger ?? false,
  }
  confirmState.open = true
  return new Promise<boolean>((resolve) => {
    confirmState.resolve = resolve
  })
}

export function settleConfirm(value: boolean) {
  const r = confirmState.resolve
  confirmState.resolve = null
  confirmState.open = false
  r?.(value)
}

export function useConfirm() {
  return confirm
}
