// Tema yang dipilih pengunjung di landing page (/daftar?tema=<slug>).
// Disimpan agar bisa disorot saat customer membuka tab Tema undangan baru — tidak dipilih otomatis.

const KEY = 'pending_theme'
const SLUG_RE = /^[a-z0-9][a-z0-9-]{0,62}$/

export function savePendingTheme(slug: unknown) {
  if (typeof slug !== 'string') return
  const clean = slug.trim().toLowerCase()
  if (!SLUG_RE.test(clean)) return
  try {
    localStorage.setItem(KEY, clean)
  } catch {
    /* storage tidak tersedia (mode privat) — abaikan */
  }
}

export function getPendingTheme(): string | null {
  try {
    const v = localStorage.getItem(KEY)
    return v && SLUG_RE.test(v) ? v : null
  } catch {
    return null
  }
}

export function clearPendingTheme() {
  try {
    localStorage.removeItem(KEY)
  } catch {
    /* abaikan */
  }
}

/** "jawa-sogan" → "Jawa Sogan" (label sementara sebelum data tema dimuat). */
export function prettySlug(slug: string) {
  return slug
    .split('-')
    .filter(Boolean)
    .map((w) => w[0].toUpperCase() + w.slice(1))
    .join(' ')
}
