import type { EventItem, InvitationContent, Person } from './types'

export function emptyPerson(): Person {
  return {
    full_name: '',
    nickname: '',
    photo: '',
    child_order: '',
    father: '',
    mother: '',
    father_deceased: false,
    mother_deceased: false,
    instagram: '',
  }
}

export function newEvent(name = ''): EventItem {
  return {
    id: crypto.randomUUID(),
    name,
    date: '',
    start_time: '',
    end_time: '',
    timezone: 'WIB',
    venue: '',
    address: '',
    maps_url: '',
    note: '',
  }
}

/** Konten kosong lengkap — backend bisa mengirim konten parsial, gabungkan dengan normalizeContent. */
export function emptyContent(): InvitationContent {
  return {
    event_type: 'pernikahan',
    religion: 'islam',
    couple_order: 'groom_first',
    cover: { title: '', photo: '', background: '' },
    opening: { greeting: '', text: '' },
    quote: { text: '', source: '' },
    groom: emptyPerson(),
    bride: emptyPerson(),
    events: [newEvent('Akad Nikah'), newEvent('Resepsi')],
    love_story: [],
    gallery: { photos: [], video_url: '' },
    live_stream: { url: '', platform: '', note: '' },
    gift: { enabled: false, confirmation_enabled: false, text: '', accounts: [], address: { recipient: '', phone: '', address: '' } },
    rsvp: { enabled: true, max_pax: 2, show_wishes: true },
    closing: { text: '', sign_off: '', from: '', family: [] },
    music: { enabled: false, url: '', title: '', start_at: 0 },
    share: { whatsapp_template: '' },
  }
}

function isObj(v: unknown): v is Record<string, unknown> {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

function merge<T>(base: T, patch: unknown): T {
  if (!isObj(base) || !isObj(patch)) return (patch ?? base) as T
  const out: Record<string, unknown> = { ...base }
  for (const [k, v] of Object.entries(patch)) {
    if (v === null || v === undefined) continue
    out[k] = isObj(out[k]) && isObj(v) ? merge(out[k], v) : v
  }
  return out as T
}

export function normalizeContent(raw: unknown): InvitationContent {
  const c = merge(emptyContent(), raw)
  if (isObj(raw) && !Array.isArray((raw as Record<string, unknown>).events)) c.events = emptyContent().events
  return c
}

export const RELIGIONS = [
  { value: 'islam', label: 'Islam' },
  { value: 'kristen', label: 'Kristen' },
  { value: 'katolik', label: 'Katolik' },
  { value: 'hindu', label: 'Hindu' },
  { value: 'buddha', label: 'Buddha' },
  { value: 'konghucu', label: 'Konghucu' },
  { value: 'umum', label: 'Umum / Netral' },
] as const

export const EVENT_NAME_SUGGESTIONS = ['Akad Nikah', 'Pemberkatan', 'Resepsi', 'Ngunduh Mantu', 'Unduh Mantu', 'Temu Manten', 'Walimatul Ursy']

export const BANK_SUGGESTIONS = ['BCA', 'BRI', 'BNI', 'Mandiri', 'BSI', 'CIMB Niaga', 'Permata', 'BTN', 'Danamon', 'Jago', 'SeaBank', 'Blu']
export const EWALLET_SUGGESTIONS = ['DANA', 'OVO', 'GoPay', 'ShopeePay', 'LinkAja']

export function waLink(phone: string, text: string) {
  const p = phone.replace(/\D/g, '').replace(/^0/, '62')
  return `https://wa.me/${p}?text=${encodeURIComponent(text)}`
}

export function fillShareTemplate(tpl: string, name: string, link: string) {
  return tpl.replaceAll('{nama}', name).replaceAll('{link}', link)
}
