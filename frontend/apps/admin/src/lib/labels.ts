import type { EventType, InvitationStatus, OrderStatus, Role, SubscriptionStatus } from '@undangan/shared'

export type Tone = 'neutral' | 'accent' | 'success' | 'warning' | 'danger'

export const ORDER_STATUS: Record<OrderStatus, { label: string; tone: Tone }> = {
  awaiting_confirmation: { label: 'Menunggu konfirmasi', tone: 'warning' },
  awaiting_payment: { label: 'Menunggu pembayaran', tone: 'accent' },
  paid: { label: 'Lunas', tone: 'success' },
  rejected: { label: 'Ditolak', tone: 'danger' },
  expired: { label: 'Kedaluwarsa', tone: 'neutral' },
}

export const SUBSCRIPTION_STATUS: Record<SubscriptionStatus, { label: string; tone: Tone }> = {
  active: { label: 'Aktif', tone: 'success' },
  grace: { label: 'Masa tenggang', tone: 'warning' },
  expired: { label: 'Kedaluwarsa', tone: 'danger' },
  cancelled: { label: 'Dibatalkan', tone: 'neutral' },
}

export const INVITATION_STATUS: Record<InvitationStatus, { label: string; tone: Tone }> = {
  draft: { label: 'Draf', tone: 'neutral' },
  published: { label: 'Terbit', tone: 'success' },
  suspended: { label: 'Ditangguhkan', tone: 'danger' },
}

export const ROLE_LABEL: Record<Role, string> = {
  super_admin: 'Super admin',
  customer: 'Customer',
}

export const EVENT_TYPE_LABEL: Record<EventType, string> = {
  pernikahan: 'Pernikahan',
  ngunduh_mantu: 'Ngunduh Mantu',
}
