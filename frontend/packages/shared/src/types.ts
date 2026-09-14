// Kontrak data — harus sama dengan docs/SPEC.md dan backend Go.

export type Role = 'super_admin' | 'customer'
export type Portal = 'admin' | 'customer'

export interface User {
  id: string
  name: string
  email: string
  phone: string
  role: Role
  is_suspended: boolean
  created_at: string
}

export interface Paginated<T> {
  items: T[]
  total: number
  page: number
  per_page: number
}

export interface ApiErrorBody {
  error: { code: string; message: string; fields?: Record<string, string> }
}

// ---------- Undangan ----------

export type Religion = 'islam' | 'kristen' | 'katolik' | 'hindu' | 'buddha' | 'konghucu' | 'umum'
export type EventType = 'pernikahan' | 'ngunduh_mantu'
export type Timezone = 'WIB' | 'WITA' | 'WIT'

export interface Person {
  full_name: string
  nickname: string
  photo: string
  child_order: string
  father: string
  mother: string
  father_deceased: boolean
  mother_deceased: boolean
  instagram: string
}

export interface EventItem {
  id: string
  name: string
  date: string // YYYY-MM-DD
  start_time: string // HH:mm
  end_time: string // '' = Selesai
  timezone: Timezone
  venue: string
  address: string
  maps_url: string
  note: string
}

export interface StoryItem {
  date: string
  title: string
  text: string
  photo: string
}

export interface GiftAccount {
  type: 'bank' | 'ewallet'
  provider: string
  number: string
  holder: string
}

export interface InvitationContent {
  event_type: EventType
  religion: Religion
  couple_order: 'groom_first' | 'bride_first'
  cover: { title: string; photo: string; background: string }
  opening: { greeting: string; text: string }
  quote: { text: string; source: string }
  groom: Person
  bride: Person
  events: EventItem[]
  love_story: StoryItem[]
  gallery: { photos: string[]; video_url: string }
  live_stream: { url: string; platform: string; note: string }
  gift: {
    enabled: boolean
    /** Tamu boleh mengirim bukti transfer / kado dari halaman undangan. */
    confirmation_enabled: boolean
    text: string
    accounts: GiftAccount[]
    address: { recipient: string; phone: string; address: string }
  }
  rsvp: { enabled: boolean; max_pax: number; show_wishes: boolean }
  closing: { text: string; sign_off: string; from: string; family: string[] }
  music: { enabled: boolean; url: string; title: string; start_at: number }
  share: { whatsapp_template: string }
}

export type InvitationStatus = 'draft' | 'published' | 'suspended'

export interface InvitationSummary {
  id: string
  user_id: string
  user_name: string
  user_email: string
  title: string // "Raka & Nadia" atau "(belum diisi)"
  theme: string | null
  theme_locked: boolean
  subdomain: string | null
  url: string | null // https://raka-nadia.<base>
  status: InvitationStatus
  event_date: string | null // tanggal acara pertama
  guest_count: number
  created_at: string
  updated_at: string
}

export type AccessMode = 'public' | 'guest_only'

export interface InvitationSettings {
  access_mode: AccessMode
  checkin_enabled: boolean
  checkin_pin_set: boolean
  /** Paket pemilik mengizinkan fitur check-in (admin tetap bisa mengaktifkan). */
  checkin_available: boolean
  /** Path stasiun check-in relatif portal customer: /checkin/<id> */
  checkin_station_path: string
  /** Paket pemilik mengizinkan domain sendiri (admin tetap bisa mengatur). */
  custom_domain_available: boolean
}

export type DnsRecordType = 'CNAME' | 'A' | 'TXT'

export interface DnsRecord {
  type: DnsRecordType
  /** Nama/host record, mis. "www" atau "@" atau "_undangan-verify.www". */
  name: string
  value: string
  /** Penjelasan singkat fungsi record. */
  purpose: string
}

export interface CustomDomain {
  hostname: string // www.rakadannadia.com
  verified: boolean
  verified_at: string | null
  dns: DnsRecord[]
}

export interface Invitation extends InvitationSummary {
  content: InvitationContent
  settings: InvitationSettings
  quota: { max_guests: number; used_guests: number }
  subscription_status: SubscriptionStatus | null
  /** Domain sendiri milik customer; null bila belum diatur. */
  custom_domain: CustomDomain | null
}

export interface ThemeInfo {
  slug: string
  name: string
  description: string
  tags: string[]
  colors: string[]
  fonts: string[]
  is_active: boolean
  is_premium: boolean
  sort_order: number
  preview_url: string // /_preview/<slug>
  category: ThemeCategory | ''
  thumbnail_url: string // '' bila belum ada
}

export type ThemeCategory = 'adat' | 'islami' | 'floral' | 'modern' | 'elegan' | 'rustic' | 'pastel' | 'retro'

export const THEME_CATEGORIES: { value: ThemeCategory; label: string }[] = [
  { value: 'adat', label: 'Adat Nusantara' },
  { value: 'islami', label: 'Islami' },
  { value: 'floral', label: 'Floral & Botanical' },
  { value: 'modern', label: 'Modern & Minimalis' },
  { value: 'elegan', label: 'Elegan & Mewah' },
  { value: 'rustic', label: 'Rustic & Boho' },
  { value: 'pastel', label: 'Pastel & Manis' },
  { value: 'retro', label: 'Retro & Unik' },
]

// ---------- Tamu & ucapan ----------

export interface Guest {
  id: string
  name: string
  phone: string
  group_name: string
  pax: number
  code: string // passcode 6 karakter (QR & check-in)
  slug: string // link personal: <url>/<slug>
  link: string | null // null jika undangan belum punya subdomain
  opened_at: string | null
  checked_in_at: string | null
  checked_in_pax: number | null
  created_at: string
}

export interface GuestInput {
  name: string
  phone: string
  group_name: string
  pax: number
}

export type ImportRowStatus = 'ok' | 'duplicate' | 'error'

export interface ImportRow {
  row: number
  name: string
  phone: string
  group_name: string
  pax: number
  status: ImportRowStatus
  error?: string
}

export interface ImportResult {
  rows: ImportRow[]
  summary: { ok: number; duplicate: number; error: number }
  inserted: number
  dry_run: boolean
}

export type Attendance = 'hadir' | 'tidak' | 'ragu'

export interface Wish {
  id: string
  name: string
  attendance: Attendance
  pax: number
  message: string
  guest_id: string | null
  is_hidden: boolean
  created_at: string
}

export interface WishList extends Paginated<Wish> {
  stats: { hadir: number; tidak: number; ragu: number; total_pax: number }
}

// ---------- Paket, order, langganan ----------

export interface Plan {
  id: string
  name: string
  description: string
  price: number
  duration_days: number
  grace_days: number
  max_invitations: number
  max_guests: number
  allow_custom_domain: boolean
  allow_checkin: boolean
  is_active: boolean
  sort_order: number
}

export type OrderStatus = 'awaiting_payment' | 'awaiting_confirmation' | 'paid' | 'rejected' | 'expired'

export interface Order {
  id: string
  code: string // INV-20260913-AB12
  user_id: string
  user_name: string
  user_email: string
  plan_id: string
  plan_name: string
  price: number
  unique_code: number
  amount: number // price + unique_code
  status: OrderStatus
  proof_url: string | null // /api/v1/orders/{id}/proof
  proof_uploaded_at: string | null
  reject_reason: string
  reviewed_by_name: string | null
  reviewed_at: string | null
  expires_at: string
  created_at: string
}

export type SubscriptionStatus = 'active' | 'grace' | 'expired' | 'cancelled'

export interface Subscription {
  id: string
  user_id: string
  user_name: string
  user_email: string
  plan_id: string
  plan_name: string
  status: SubscriptionStatus
  starts_at: string
  ends_at: string
  grace_ends_at: string
  max_invitations: number
  max_guests: number
  allow_checkin: boolean
  created_at: string
}

export interface PaymentSettings {
  qris_image: string
  merchant_name: string
  instructions: string
  admin_whatsapp: string // 628xx
}

export interface MusicTrack {
  id: string
  title: string
  artist: string
  url: string
  duration_seconds: number
  created_at: string
}

export interface AdminStats {
  users: number
  active_subscriptions: number
  grace_subscriptions: number
  pending_orders: number
  invitations: number
  published_invitations: number
  revenue_this_month: number
  guests: number
}

export interface AuditLog {
  id: number
  actor_name: string | null
  action: string // "order.approve"
  target: string // "order:uuid"
  meta: Record<string, unknown> | null
  created_at: string
}

// ---------- Check-in & kehadiran ----------

export interface AttendanceSummary {
  invited_guests: number
  invited_pax: number
  rsvp_hadir: number
  rsvp_pax: number
  checked_in_guests: number
  checked_in_pax: number
  not_checked_in_guests: number
}

export interface AttendanceItem {
  guest_id: string
  name: string
  group_name: string
  phone: string
  pax: number
  code: string
  rsvp_attendance: Attendance | null
  rsvp_pax: number | null
  checked_in_at: string | null
  checked_in_pax: number | null
  checked_in_by: string
}

export interface AttendanceReport extends Paginated<AttendanceItem> {
  summary: AttendanceSummary
}

export interface CheckinSession {
  token: string
  expires_at: string
  invitation: { id: string; title: string; event_date: string | null }
}

export interface CheckinSummary {
  invitation: { id: string; title: string; event_date: string | null }
  checkin_enabled: boolean
  summary: AttendanceSummary
}

export interface CheckinGuest {
  id: string
  name: string
  group_name: string
  pax: number
  code: string
  checked_in_at: string | null
  checked_in_pax: number | null
  checked_in_by: string
}

export type CheckinResultStatus = 'checked_in' | 'already' | 'not_found' | 'disabled'

export interface CheckinResult {
  status: CheckinResultStatus
  message: string
  guest: CheckinGuest | null
}

export interface CheckinLog {
  id: number
  guest_name: string | null
  input: string
  result: CheckinResultStatus
  pax: number | null
  station: string
  created_at: string
}

// ---------- Konfirmasi hadiah ----------

export type GiftType = 'transfer' | 'kado'

export interface GiftConfirmation {
  id: string
  guest_id: string | null
  guest_name: string | null // nama di daftar tamu bila dikirim dari link personal
  name: string
  type: GiftType
  account_label: string
  amount: number | null
  message: string
  proof_url: string // /api/v1/invitations/{id}/gifts/{gid}/proof
  is_verified: boolean
  verified_at: string | null
  created_at: string
}

export interface GiftList extends Paginated<GiftConfirmation> {
  summary: { count: number; verified_count: number; total_amount: number; verified_amount: number }
}
