/** Nama merek yang tampil di portal. */
export const APP_NAME = import.meta.env.VITE_APP_NAME?.trim() || 'Undangin'

/** Landing page marketing (tombol "Kembali ke beranda"). */
export const LANDING_URL = (import.meta.env.VITE_LANDING_URL?.trim() || 'http://localhost:8080').replace(/\/+$/, '') || '/'
