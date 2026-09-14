<script setup lang="ts">
import {
  api,
  compressImage,
  copyText,
  errorMessage,
  formatDate,
  formatRupiah,
  toast,
  waLink,
  type Order,
  type PaymentSettings,
} from '@undangan/shared'
import UiBadge from '@undangan/shared/components/ui/UiBadge.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiDialog from '@undangan/shared/components/ui/UiDialog.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ORDER_STATUS } from '../lib/labels'

const props = defineProps<{ id: string }>()

const order = ref<Order | null>(null)
const settings = ref<PaymentSettings | null>(null)
const loading = ref(true)
const error = ref('')

async function load(silent = false) {
  if (!silent) loading.value = true
  error.value = ''
  try {
    order.value = await api.get<Order>(`/me/orders/${props.id}`)
    if (!settings.value && (order.value.status === 'awaiting_payment' || order.value.status === 'rejected')) {
      settings.value = await api.get<PaymentSettings>('/payment-settings').catch(() => null)
    }
  } catch (e) {
    if (!silent) error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

watch(() => props.id, () => load(), { immediate: true })

// ---------- Status & stepper ----------
const status = computed(() => order.value?.status)
const needsPayment = computed(() => status.value === 'awaiting_payment' || status.value === 'rejected')
const steps = ['Pesan paket', 'Bayar & kirim bukti', 'Verifikasi admin', 'Paket aktif']
const stepIndex = computed(() => {
  switch (status.value) {
    case 'awaiting_confirmation':
      return 2
    case 'paid':
      return 4
    default:
      return 1
  }
})

// ---------- Nominal ----------
const amountParts = computed(() => {
  const o = order.value
  if (!o) return { head: '', tail: '' }
  const s = formatRupiah(o.amount)
  const digits = String(o.unique_code).length
  // Sorot digit terakhir sesuai panjang kode unik (maks. 3 digit).
  const n = Math.min(3, Math.max(1, digits))
  let count = 0
  let cut = s.length
  for (let i = s.length - 1; i >= 0; i--) {
    if (/\d/.test(s[i]!)) {
      count++
      if (count === n) {
        cut = i
        break
      }
    }
  }
  return { head: s.slice(0, cut), tail: s.slice(cut) }
})

async function copyAmount() {
  if (!order.value) return
  if (await copyText(String(order.value.amount))) toast.success('Nominal disalin')
}

async function copyCode() {
  if (order.value && (await copyText(order.value.code))) toast.success('Kode order disalin')
}

// ---------- Hitung mundur ----------
const now = ref(Date.now())
const timer = setInterval(() => {
  now.value = Date.now()
}, 1000)
const remainingMs = computed(() => (order.value ? new Date(order.value.expires_at).getTime() - now.value : 0))
const countdown = computed(() => {
  const ms = Math.max(0, remainingMs.value)
  const h = Math.floor(ms / 3_600_000)
  const m = Math.floor((ms % 3_600_000) / 60_000)
  const s = Math.floor((ms % 60_000) / 1000)
  return [h, m, s].map((x) => String(x).padStart(2, '0')).join(':')
})
let expiredReloaded = false
watch(remainingMs, (ms) => {
  if (ms <= 0 && status.value === 'awaiting_payment' && !expiredReloaded) {
    expiredReloaded = true
    load(true)
  }
})

// Polling saat menunggu verifikasi.
const poll = setInterval(() => {
  if (status.value === 'awaiting_confirmation' && document.visibilityState === 'visible') load(true)
}, 20_000)

onBeforeUnmount(() => {
  clearInterval(timer)
  clearInterval(poll)
  if (proofPreview.value) URL.revokeObjectURL(proofPreview.value)
})

// ---------- Upload bukti ----------
const fileInput = ref<HTMLInputElement | null>(null)
const proofFile = ref<File | null>(null)
const proofPreview = ref('')
const uploading = ref(false)
const MAX_MB = 5

function onProof(e: Event) {
  const el = e.target as HTMLInputElement
  const file = el.files?.[0]
  el.value = ''
  if (!file) return
  if (!/^image\/(jpeg|png|webp)$/.test(file.type)) {
    toast.error('Bukti harus berupa gambar JPG, PNG, atau WebP')
    return
  }
  if (file.size > MAX_MB * 1024 * 1024) {
    toast.error(`Ukuran bukti maksimal ${MAX_MB} MB`)
    return
  }
  if (proofPreview.value) URL.revokeObjectURL(proofPreview.value)
  proofFile.value = file
  proofPreview.value = URL.createObjectURL(file)
}

function clearProof() {
  if (proofPreview.value) URL.revokeObjectURL(proofPreview.value)
  proofFile.value = null
  proofPreview.value = ''
}

async function submitProof() {
  if (!proofFile.value || !order.value) return
  uploading.value = true
  try {
    const file = await compressImage(proofFile.value, { maxSize: 2000, quality: 0.85, skipBelowBytes: 1024 * 1024 })
    order.value = await api.upload<Order>(`/me/orders/${order.value.id}/proof`, file)
    clearProof()
    toast.success('Bukti pembayaran terkirim. Admin akan segera memverifikasi.')
    window.scrollTo({ top: 0, behavior: 'smooth' })
  } catch (e) {
    toast.error(e)
  } finally {
    uploading.value = false
  }
}

const waHref = computed(() => {
  const o = order.value
  const phone = settings.value?.admin_whatsapp
  if (!o || !phone) return ''
  return waLink(
    phone,
    `Halo Admin, saya sudah melakukan pembayaran untuk order ${o.code} (paket ${o.plan_name}) sebesar ${formatRupiah(o.amount)}. Mohon dikonfirmasi. Terima kasih.`,
  )
})

const qrisZoom = ref(false)
const proofZoom = ref(false)
</script>

<template>
  <div class="mx-auto max-w-3xl px-4 pt-2 md:px-6 md:pt-10">
    <RouterLink to="/pembayaran" class="mb-4 inline-flex items-center gap-1 text-sm text-muted hover:text-ink">
      <UiIcon name="arrow-left" :size="15" /> Riwayat pembayaran
    </RouterLink>

    <div v-if="loading" class="flex items-center justify-center gap-2 py-24 text-sm text-muted"><UiSpinner /> Memuat order…</div>
    <div v-else-if="error || !order" class="card p-8 text-center">
      <p class="text-sm text-danger">{{ error || 'Order tidak ditemukan' }}</p>
      <UiButton variant="secondary" size="sm" class="mt-3" @click="load()"><UiIcon name="refresh" :size="14" /> Coba lagi</UiButton>
    </div>

    <template v-else>
      <!-- Ringkasan -->
      <div class="card mb-4 p-5 sm:p-6">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p class="text-xs tracking-wider text-muted uppercase">Order paket</p>
            <h1 class="heading text-3xl sm:text-4xl">{{ order.plan_name }}</h1>
            <button type="button" class="mt-1 inline-flex items-center gap-1 font-mono text-xs text-muted hover:text-ink" @click="copyCode">
              {{ order.code }} <UiIcon name="copy" :size="12" />
            </button>
          </div>
          <UiBadge :tone="ORDER_STATUS[order.status].tone">{{ ORDER_STATUS[order.status].label }}</UiBadge>
        </div>

        <!-- Stepper -->
        <ol v-if="order.status !== 'expired'" class="mt-6 grid grid-cols-4 gap-1">
          <li v-for="(s, i) in steps" :key="s" class="flex flex-col items-center text-center">
            <div class="flex w-full items-center">
              <span class="h-0.5 flex-1" :class="i === 0 ? 'bg-transparent' : i <= stepIndex ? 'bg-accent' : 'bg-line'" />
              <span
                class="flex size-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
                :class="[
                  i < stepIndex
                    ? 'bg-accent text-accent-ink'
                    : i === stepIndex && order.status === 'rejected'
                      ? 'bg-danger text-accent-ink'
                      : i === stepIndex
                        ? 'bg-accent/15 text-accent ring-2 ring-accent'
                        : 'bg-surface-2 text-muted',
                ]"
              >
                <UiIcon v-if="i < stepIndex" name="check" :size="14" :stroke-width="3" />
                <UiIcon v-else-if="i === stepIndex && order.status === 'rejected'" name="x" :size="14" :stroke-width="3" />
                <template v-else>{{ i + 1 }}</template>
              </span>
              <span class="h-0.5 flex-1" :class="i === steps.length - 1 ? 'bg-transparent' : i < stepIndex ? 'bg-accent' : 'bg-line'" />
            </div>
            <span class="mt-2 text-[11px] leading-tight sm:text-xs" :class="i <= stepIndex ? 'font-medium text-ink' : 'text-muted'">{{ s }}</span>
          </li>
        </ol>
      </div>

      <!-- Ditolak -->
      <div v-if="order.status === 'rejected'" class="mb-4 flex gap-3 rounded-2xl border border-danger/25 bg-danger/10 p-4 text-sm">
        <UiIcon name="x-circle" :size="20" class="shrink-0 text-danger" />
        <div>
          <p class="font-semibold text-danger">Pembayaran ditolak</p>
          <p class="mt-0.5 text-ink">{{ order.reject_reason || 'Bukti pembayaran tidak dapat diverifikasi.' }}</p>
          <p class="mt-1 text-xs text-muted">Silakan periksa kembali dan unggah ulang bukti pembayaran yang benar.</p>
        </div>
      </div>

      <!-- Kedaluwarsa -->
      <div v-if="order.status === 'expired'" class="card p-8 text-center">
        <div class="mx-auto mb-3 flex size-12 items-center justify-center rounded-full bg-surface-2 text-muted"><UiIcon name="clock" :size="22" /></div>
        <p class="heading text-2xl">Order kedaluwarsa</p>
        <p class="mt-1 text-sm text-muted">Batas waktu pembayaran telah lewat ({{ formatDate(order.expires_at, true) }}). Silakan buat order baru.</p>
        <UiButton class="mt-4" href="/paket" @click.prevent="$router.push('/paket')">Pilih paket lagi</UiButton>
      </div>

      <!-- Lunas -->
      <div v-else-if="order.status === 'paid'" class="card overflow-hidden p-8 text-center">
        <div class="mx-auto mb-4 flex size-16 items-center justify-center rounded-full bg-success/12 text-success">
          <UiIcon name="check-circle" :size="32" />
        </div>
        <p class="heading text-3xl">Pembayaran berhasil!</p>
        <p class="mx-auto mt-2 max-w-md text-sm text-muted">
          Paket <strong class="text-ink">{{ order.plan_name }}</strong> sudah aktif<template v-if="order.reviewed_at"> sejak {{ formatDate(order.reviewed_at, true) }}</template>.
          Saatnya membuat undangan yang indah.
        </p>
        <div class="mt-5 flex flex-wrap justify-center gap-2">
          <UiButton href="/undangan" @click.prevent="$router.push('/undangan')"><UiIcon name="plus" :size="16" /> Buat undangan</UiButton>
          <UiButton variant="secondary" href="/" @click.prevent="$router.push('/')">Ke beranda</UiButton>
        </div>
      </div>

      <!-- Menunggu verifikasi -->
      <div v-else-if="order.status === 'awaiting_confirmation'" class="card p-6 sm:p-8">
        <div class="flex flex-col items-center text-center">
          <div class="relative mb-4 flex size-16 items-center justify-center rounded-full bg-accent/10 text-accent">
            <span class="absolute inset-0 animate-ping rounded-full bg-accent/10" />
            <UiIcon name="clock" :size="28" />
          </div>
          <p class="heading text-3xl">Menunggu verifikasi</p>
          <p class="mt-1 max-w-md text-sm text-muted">
            Bukti pembayaran sudah kami terima<template v-if="order.proof_uploaded_at"> ({{ formatDate(order.proof_uploaded_at, true) }})</template>. Admin akan
            memverifikasi secepatnya — halaman ini diperbarui otomatis.
          </p>
        </div>
        <div class="mt-6 flex items-center gap-4 rounded-2xl bg-surface-2 p-4">
          <button
            v-if="order.proof_url"
            type="button"
            class="size-20 shrink-0 overflow-hidden rounded-xl border border-line bg-surface"
            aria-label="Lihat bukti"
            @click="proofZoom = true"
          >
            <img :src="order.proof_url" alt="Bukti pembayaran" class="h-full w-full object-cover" />
          </button>
          <div class="min-w-0 text-sm">
            <p class="text-muted">Nominal</p>
            <p class="text-lg font-semibold text-ink">{{ formatRupiah(order.amount) }}</p>
          </div>
        </div>
        <div class="mt-4 flex flex-wrap justify-center gap-2">
          <UiButton variant="secondary" size="sm" @click="load()"><UiIcon name="refresh" :size="14" /> Perbarui status</UiButton>
        </div>
      </div>

      <!-- Perlu bayar (menunggu / ditolak) -->
      <div v-if="needsPayment" class="space-y-4">
        <div class="card overflow-hidden">
          <div v-if="order.status === 'awaiting_payment'" class="flex items-center justify-between gap-3 border-b border-line bg-surface-2/60 px-5 py-3 text-sm">
            <span class="text-muted">Bayar sebelum {{ formatDate(order.expires_at, true) }}</span>
            <span class="flex items-center gap-1.5 font-mono font-semibold" :class="remainingMs < 3_600_000 ? 'text-danger' : 'text-ink'">
              <UiIcon name="clock" :size="15" /> {{ countdown }}
            </span>
          </div>

          <div class="grid gap-6 p-5 sm:p-6 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
            <!-- QRIS -->
            <div class="flex flex-col items-center">
              <p class="mb-2 text-xs tracking-wider text-muted uppercase">Scan QRIS</p>
              <button
                v-if="settings?.qris_image"
                type="button"
                class="w-full max-w-72 overflow-hidden rounded-2xl border border-line bg-surface p-3 shadow-soft"
                aria-label="Perbesar QRIS"
                @click="qrisZoom = true"
              >
                <img :src="settings.qris_image" alt="Kode QRIS" class="aspect-square w-full object-contain" />
              </button>
              <div v-else class="flex aspect-square w-full max-w-72 flex-col items-center justify-center gap-2 rounded-2xl border border-dashed border-line text-sm text-muted">
                <UiIcon name="qr" :size="32" /> QRIS belum tersedia
              </div>
              <p v-if="settings?.merchant_name" class="mt-3 text-sm font-semibold text-ink">{{ settings.merchant_name }}</p>
              <p class="text-xs text-muted">Bisa dibayar dengan semua e-wallet & m-banking</p>
            </div>

            <!-- Nominal -->
            <div class="flex flex-col">
              <p class="text-xs tracking-wider text-muted uppercase">Total transfer</p>
              <div class="mt-1 flex items-center gap-2">
                <p class="text-3xl font-semibold tracking-tight text-ink sm:text-4xl">
                  {{ amountParts.head }}<span class="rounded-md bg-accent/15 px-0.5 text-accent">{{ amountParts.tail }}</span>
                </p>
                <button type="button" class="rounded-lg border border-line p-2 text-muted hover:bg-surface-2 hover:text-ink" aria-label="Salin nominal" @click="copyAmount">
                  <UiIcon name="copy" :size="16" />
                </button>
              </div>
              <div class="mt-3 flex items-start gap-2 rounded-xl bg-warning/10 px-3 py-2.5 text-sm text-ink">
                <UiIcon name="alert" :size="16" class="mt-0.5 shrink-0 text-warning" />
                <p>Transfer sesuai nominal persis agar mudah diverifikasi. Tiga digit terakhir adalah kode unik.</p>
              </div>
              <dl class="mt-4 space-y-1.5 text-sm">
                <div class="flex justify-between gap-2"><dt class="text-muted">Harga paket</dt><dd class="text-ink">{{ formatRupiah(order.price) }}</dd></div>
                <div class="flex justify-between gap-2"><dt class="text-muted">Kode unik</dt><dd class="text-ink">+{{ order.unique_code }}</dd></div>
                <div class="flex justify-between gap-2 border-t border-line pt-1.5 font-semibold"><dt class="text-ink">Total</dt><dd class="text-ink">{{ formatRupiah(order.amount) }}</dd></div>
              </dl>
              <div v-if="settings?.instructions" class="mt-4 rounded-xl bg-surface-2 p-3 text-sm leading-relaxed whitespace-pre-line text-muted">
                {{ settings.instructions }}
              </div>
            </div>
          </div>
        </div>

        <!-- Upload bukti -->
        <div class="card p-5 sm:p-6">
          <h2 class="heading text-2xl">{{ order.status === 'rejected' ? 'Unggah ulang bukti pembayaran' : 'Unggah bukti pembayaran' }}</h2>
          <p class="mt-1 text-sm text-muted">Screenshot / foto bukti transfer yang menampilkan nominal & waktu. JPG, PNG, atau WebP maks. {{ MAX_MB }} MB.</p>
          <input ref="fileInput" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="onProof" />

          <div v-if="proofPreview" class="mt-4 flex flex-col gap-4 sm:flex-row sm:items-start">
            <img :src="proofPreview" alt="Pratinjau bukti" class="max-h-80 w-full rounded-xl border border-line object-contain sm:w-56" />
            <div class="flex flex-1 flex-col gap-2">
              <p class="truncate text-sm text-ink">{{ proofFile?.name }}</p>
              <div class="flex flex-wrap gap-2">
                <UiButton :loading="uploading" @click="submitProof"><UiIcon name="send" :size="15" /> Kirim bukti</UiButton>
                <UiButton variant="ghost" :disabled="uploading" @click="clearProof">Ganti file</UiButton>
              </div>
            </div>
          </div>
          <button
            v-else
            type="button"
            class="mt-4 flex w-full flex-col items-center gap-2 rounded-2xl border-2 border-dashed border-line px-4 py-8 text-sm text-muted transition hover:border-accent hover:text-accent"
            @click="fileInput?.click()"
          >
            <UiIcon name="upload" :size="26" />
            <span class="font-medium">Pilih foto bukti transfer</span>
          </button>

          <div v-if="waHref" class="mt-5 flex flex-col items-center gap-2 border-t border-line pt-5 text-center sm:flex-row sm:text-left">
            <p class="flex-1 text-sm text-muted">Sudah bayar? Kabari admin agar diverifikasi lebih cepat.</p>
            <UiButton variant="secondary" :href="waHref" target="_blank" rel="noopener">
              <UiIcon name="message" :size="16" class="text-success" /> Konfirmasi via WhatsApp
            </UiButton>
          </div>
        </div>
      </div>

      <UiDialog v-model:open="qrisZoom" title="QRIS" size="sm">
        <img v-if="settings?.qris_image" :src="settings.qris_image" alt="Kode QRIS" class="w-full rounded-xl" />
        <p class="mt-3 text-center text-sm text-ink">{{ settings?.merchant_name }} · <strong>{{ formatRupiah(order.amount) }}</strong></p>
      </UiDialog>
      <UiDialog v-model:open="proofZoom" title="Bukti pembayaran" size="md">
        <img v-if="order.proof_url" :src="order.proof_url" alt="Bukti pembayaran" class="w-full rounded-xl" />
      </UiDialog>
    </template>
  </div>
</template>
