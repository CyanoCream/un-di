/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string
  /** Domain dasar undangan untuk akhiran subdomain, mis. "undangan.id". */
  readonly VITE_BASE_DOMAIN?: string
  /** URL landing page marketing, mis. "https://undangin.id". */
  readonly VITE_LANDING_URL?: string
  /** Nama merek, bawaan "Undangin". */
  readonly VITE_APP_NAME?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
