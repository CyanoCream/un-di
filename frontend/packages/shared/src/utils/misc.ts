import { ApiError } from '../api'
import type { InvitationContent } from '../types'

/** Pesan error yang ramah untuk ditampilkan ke pengguna. */
export function errorMessage(e: unknown, fallback = 'Terjadi kesalahan. Coba lagi.') {
  if (e instanceof ApiError) return e.message
  if (e instanceof TypeError) return 'Tidak dapat terhubung ke server. Periksa koneksi internet Anda.'
  if (e instanceof Error && e.message) return e.message
  return fallback
}

/** Salin teks ke clipboard (fallback execCommand untuk konteks non-HTTPS). */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    /* fallback di bawah */
  }
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}

export function debounce<A extends unknown[]>(fn: (...args: A) => void, wait: number) {
  let t: ReturnType<typeof setTimeout> | undefined
  const d = (...args: A) => {
    if (t) clearTimeout(t)
    t = setTimeout(() => fn(...args), wait)
  }
  d.cancel = () => t && clearTimeout(t)
  return d
}

/** Huruf kecil, spasi/underscore → '-', buang karakter selain a-z 0-9 -. */
export function sanitizeSubdomain(v: string) {
  return v
    .toLowerCase()
    .replace(/[\s_.]+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/-{2,}/g, '-')
    .slice(0, 63)
}

/** Validasi lokal subdomain; mengembalikan pesan error atau null. */
export function validateSubdomain(v: string): string | null {
  if (!v) return 'Subdomain wajib diisi'
  if (v.length < 3) return 'Minimal 3 karakter'
  if (v.startsWith('-') || v.endsWith('-')) return 'Tidak boleh diawali atau diakhiri tanda -'
  if (!/^[a-z0-9-]+$/.test(v)) return 'Hanya huruf kecil, angka, dan tanda -'
  return null
}

export function daysUntil(iso: string | null | undefined) {
  if (!iso) return 0
  return Math.max(0, Math.ceil((new Date(iso).getTime() - Date.now()) / 86_400_000))
}

/** Template pesan WhatsApp bawaan bila customer belum mengisi. */
export function defaultWhatsappTemplate(content?: Pick<InvitationContent, 'groom' | 'bride' | 'event_type'> | null) {
  const g = content?.groom.nickname || content?.groom.full_name || ''
  const b = content?.bride.nickname || content?.bride.full_name || ''
  const couple = g && b ? ` *${g} & ${b}*` : ''
  const acara = content?.event_type === 'ngunduh_mantu' ? 'acara ngunduh mantu' : 'acara pernikahan'
  return [
    'Kepada Yth.',
    'Bapak/Ibu/Saudara/i',
    '*{nama}*',
    '',
    `Tanpa mengurangi rasa hormat, perkenankan kami mengundang Bapak/Ibu/Saudara/i untuk hadir di ${acara} kami${couple}.`,
    '',
    'Info lengkap acara dapat dilihat melalui tautan berikut:',
    '{link}',
    '',
    'Merupakan suatu kebahagiaan bagi kami apabila Bapak/Ibu/Saudara/i berkenan hadir dan memberikan doa restu.',
    '',
    'Terima kasih.',
  ].join('\n')
}
