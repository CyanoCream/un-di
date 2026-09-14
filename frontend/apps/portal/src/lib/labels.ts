import type { InvitationStatus, OrderStatus, SubscriptionStatus } from '@undangan/shared'

type Tone = 'neutral' | 'accent' | 'success' | 'warning' | 'danger'

export const ORDER_STATUS: Record<OrderStatus, { label: string; tone: Tone }> = {
  awaiting_payment: { label: 'Menunggu pembayaran', tone: 'warning' },
  awaiting_confirmation: { label: 'Menunggu verifikasi', tone: 'accent' },
  paid: { label: 'Lunas', tone: 'success' },
  rejected: { label: 'Ditolak', tone: 'danger' },
  expired: { label: 'Kedaluwarsa', tone: 'neutral' },
}

export const SUB_STATUS: Record<SubscriptionStatus, { label: string; tone: Tone }> = {
  active: { label: 'Aktif', tone: 'success' },
  grace: { label: 'Masa tenggang', tone: 'warning' },
  expired: { label: 'Berakhir', tone: 'neutral' },
  cancelled: { label: 'Dibatalkan', tone: 'neutral' },
}

export const INV_STATUS: Record<InvitationStatus, { label: string; tone: Tone }> = {
  draft: { label: 'Draf', tone: 'neutral' },
  published: { label: 'Terbit', tone: 'success' },
  suspended: { label: 'Nonaktif', tone: 'danger' },
}

export function greeting() {
  const h = new Date().getHours()
  if (h < 11) return 'Selamat pagi'
  if (h < 15) return 'Selamat siang'
  if (h < 18) return 'Selamat sore'
  return 'Selamat malam'
}

export const BASE_DOMAIN = import.meta.env.VITE_BASE_DOMAIN || undefined
