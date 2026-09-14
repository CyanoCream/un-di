<script setup lang="ts">
import { api, formatDate, formatRupiah, type Order, type Plan, type Subscription, type User } from '@undangan/shared'
import { computed, ref, watch } from 'vue'
import { confirmDialog } from '../lib/confirm'
import { ORDER_STATUS, SUBSCRIPTION_STATUS } from '../lib/labels'
import { refreshPendingOrders } from '../lib/pending'
import { toast } from '../lib/toast'
import { errMsg, relativeTime } from '../lib/util'
import AmountText from './AmountText.vue'
import AppIcon from './AppIcon.vue'
import CopyButton from './CopyButton.vue'
import ImageZoom from './ImageZoom.vue'
import UiBadge from './UiBadge.vue'
import UiDialog from './UiDialog.vue'
import UiDrawer from './UiDrawer.vue'
import UiSpinner from './UiSpinner.vue'
import UiState from './UiState.vue'

const props = defineProps<{ orderId: string | null; plans: Plan[] }>()
const emit = defineEmits<{ close: []; updated: [order: Order] }>()

const order = ref<Order | null>(null)
const loading = ref(false)
const error = ref('')
const currentSub = ref<Subscription | null>(null)
const subLoaded = ref(false)

const zoomSrc = ref<string | null>(null)
const proofFailed = ref(false)
const busy = ref<'approve' | 'reject' | null>(null)

const rejectOpen = ref(false)
const rejectReason = ref('')
const rejectError = ref('')
const REJECT_PRESETS = ['Nominal transfer tidak sesuai', 'Bukti bayar tidak terbaca', 'Pembayaran tidak ditemukan di mutasi', 'Bukti bayar bukan untuk order ini']

let seq = 0
async function load(id: string) {
  const my = ++seq
  loading.value = true
  error.value = ''
  proofFailed.value = false
  currentSub.value = null
  subLoaded.value = false
  try {
    const o = await api.get<Order>(`/admin/orders/${id}`)
    if (my !== seq) return
    order.value = o
    loadSubscription(o.user_id, my)
  } catch (e) {
    if (my === seq) error.value = errMsg(e)
  } finally {
    if (my === seq) loading.value = false
  }
}

async function loadSubscription(userId: string, my: number) {
  try {
    const res = await api.get<{ user: User; subscription: Subscription | null }>(`/admin/users/${userId}`)
    if (my === seq) currentSub.value = res.subscription
  } catch {
    /* info tambahan saja */
  } finally {
    if (my === seq) subLoaded.value = true
  }
}

watch(
  () => props.orderId,
  (id) => {
    order.value = null
    if (id) load(id)
  },
  { immediate: true },
)

const plan = computed(() => props.plans.find((p) => p.id === order.value?.plan_id) ?? null)
const status = computed(() => (order.value ? ORDER_STATUS[order.value.status] : null))
const canReview = computed(() => order.value?.status === 'awaiting_confirmation' || order.value?.status === 'awaiting_payment')
const expired = computed(() => !!order.value && order.value.status === 'awaiting_payment' && new Date(order.value.expires_at).getTime() < Date.now())

function addDays(base: Date, days: number) {
  return new Date(base.getTime() + days * 86_400_000)
}

function approveDetails(o: Order): string[] {
  const days = plan.value?.duration_days
  const out: string[] = [`Order ${o.code} ditandai Lunas (${formatRupiah(o.amount)}).`]
  const sub = currentSub.value
  if (sub && (sub.status === 'active' || sub.status === 'grace')) {
    const base = new Date(Math.max(Date.now(), new Date(sub.ends_at).getTime()))
    out.push(
      days
        ? `Langganan ${sub.plan_name} (${SUBSCRIPTION_STATUS[sub.status].label.toLowerCase()}, berakhir ${formatDate(sub.ends_at)}) diperpanjang ${days} hari → berakhir ±${formatDate(addDays(base, days).toISOString())}.`
        : `Langganan yang sedang berjalan diperpanjang sesuai durasi paket ${o.plan_name}.`,
    )
  } else {
    out.push(
      days
        ? `Langganan baru paket ${o.plan_name} aktif mulai sekarang selama ${days} hari (s.d. ±${formatDate(addDays(new Date(), days).toISOString())}).`
        : `Langganan baru paket ${o.plan_name} aktif mulai sekarang sesuai durasi paket.`,
    )
  }
  out.push('Undangan customer yang dinonaktifkan karena langganan habis akan dipulihkan.')
  if (!o.proof_url) out.push('Perhatian: customer belum mengunggah bukti bayar. Pastikan dana sudah masuk di mutasi.')
  out.push('Tindakan ini tercatat di audit log.')
  return out
}

async function approve() {
  const o = order.value
  if (!o) return
  const ok = await confirmDialog({
    title: 'Setujui pembayaran?',
    message: `Pastikan dana ${formatRupiah(o.amount)} dari ${o.user_name} sudah masuk sesuai nominal persis.`,
    details: approveDetails(o),
    confirmText: 'Setujui & aktifkan',
    tone: 'success',
  })
  if (!ok) return
  busy.value = 'approve'
  try {
    const updated = await api.post<Order>(`/admin/orders/${o.id}/approve`)
    order.value = updated
    emit('updated', updated)
    refreshPendingOrders()
    toast.success(`Order ${updated.code} disetujui. Langganan ${updated.user_name} aktif.`)
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    busy.value = null
  }
}

function openReject() {
  rejectReason.value = ''
  rejectError.value = ''
  rejectOpen.value = true
}

async function submitReject() {
  const o = order.value
  const reason = rejectReason.value.trim()
  if (!o) return
  if (reason.length < 5) {
    rejectError.value = 'Alasan wajib diisi (min. 5 karakter) — akan ditampilkan ke customer.'
    return
  }
  busy.value = 'reject'
  try {
    const updated = await api.post<Order>(`/admin/orders/${o.id}/reject`, { reason })
    order.value = updated
    rejectOpen.value = false
    emit('updated', updated)
    refreshPendingOrders()
    toast.success(`Order ${updated.code} ditolak.`)
  } catch (e) {
    rejectError.value = errMsg(e)
  } finally {
    busy.value = null
  }
}
</script>

<template>
  <UiDrawer :open="!!orderId" width="max-w-xl" @close="emit('close')">
    <template #header>
      <div class="flex min-w-0 items-center gap-2">
        <h2 class="truncate font-mono text-[15px] font-semibold">{{ order?.code ?? 'Detail order' }}</h2>
        <UiBadge v-if="status" :tone="status.tone">{{ status.label }}</UiBadge>
      </div>
    </template>

    <UiState :loading="loading" :error="error" :empty="!order" empty-title="Order tidak ditemukan" @retry="orderId && load(orderId)">
      <div v-if="order" class="space-y-5">
        <div v-if="expired" class="flex items-start gap-2 rounded-md border border-warning/30 bg-warning/10 px-3 py-2 text-xs text-warning">
          <AppIcon name="clock" :size="14" class="mt-px" /> Batas waktu pembayaran sudah lewat ({{ formatDate(order.expires_at, true) }}).
        </div>

        <!-- Nominal -->
        <section class="rounded-lg border border-line bg-canvas/60 p-4">
          <p class="text-xs text-muted">Nominal yang harus cocok di mutasi</p>
          <div class="mt-1 flex items-center gap-2">
            <p class="text-2xl font-semibold tracking-tight"><AmountText :amount="order.amount" :unique-code="order.unique_code" /></p>
            <CopyButton :text="String(order.amount)" />
          </div>
          <dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
            <dt class="text-muted">Harga paket</dt>
            <dd class="num text-right">{{ formatRupiah(order.price) }}</dd>
            <dt class="text-muted">Kode unik</dt>
            <dd class="num text-right text-accent">+{{ order.unique_code }}</dd>
          </dl>
        </section>

        <!-- Detail -->
        <section>
          <h3 class="mb-2 text-xs font-medium tracking-wider text-muted uppercase">Detail</h3>
          <dl class="grid grid-cols-[120px_1fr] gap-x-3 gap-y-2 text-[13px]">
            <dt class="text-muted">Customer</dt>
            <dd class="min-w-0">
              <RouterLink :to="`/users/${order.user_id}`" class="font-medium hover:text-accent">{{ order.user_name }}</RouterLink>
              <p class="truncate text-xs text-muted">{{ order.user_email }}</p>
            </dd>
            <dt class="text-muted">Paket</dt>
            <dd>
              {{ order.plan_name }}
              <span v-if="plan" class="text-xs text-muted">· {{ plan.duration_days }} hari</span>
            </dd>
            <dt class="text-muted">Langganan kini</dt>
            <dd>
              <span v-if="!subLoaded" class="inline-flex items-center gap-1.5 text-xs text-muted"><UiSpinner :size="12" /> memuat…</span>
              <template v-else-if="currentSub">
                <UiBadge :tone="SUBSCRIPTION_STATUS[currentSub.status]?.tone">{{ SUBSCRIPTION_STATUS[currentSub.status]?.label }}</UiBadge>
                <span class="ml-1.5 text-xs text-muted">{{ currentSub.plan_name }} · s.d. {{ formatDate(currentSub.ends_at) }}</span>
              </template>
              <span v-else class="text-xs text-muted">Belum ada langganan</span>
            </dd>
            <dt class="text-muted">Dibuat</dt>
            <dd class="num">{{ formatDate(order.created_at, true) }}</dd>
            <dt class="text-muted">Batas bayar</dt>
            <dd class="num">{{ formatDate(order.expires_at, true) }}</dd>
            <dt class="text-muted">Bukti diunggah</dt>
            <dd class="num">
              {{ formatDate(order.proof_uploaded_at, true) }}
              <span v-if="order.proof_uploaded_at" class="text-xs text-muted">({{ relativeTime(order.proof_uploaded_at) }})</span>
            </dd>
          </dl>
        </section>

        <!-- Hasil review -->
        <section
          v-if="order.reviewed_at || order.reject_reason"
          class="rounded-md border px-3 py-2.5 text-[13px]"
          :class="order.status === 'rejected' ? 'border-danger/30 bg-danger/5' : 'border-success/30 bg-success/5'"
        >
          <p class="flex items-center gap-1.5 font-medium" :class="order.status === 'rejected' ? 'text-danger' : 'text-success'">
            <AppIcon :name="order.status === 'rejected' ? 'x-circle' : 'check-circle'" :size="14" />
            {{ order.status === 'rejected' ? 'Ditolak' : 'Direview' }} oleh {{ order.reviewed_by_name ?? 'sistem' }}
          </p>
          <p class="num mt-0.5 text-xs text-muted">{{ formatDate(order.reviewed_at, true) }}</p>
          <p v-if="order.reject_reason" class="mt-1.5 text-ink/90">“{{ order.reject_reason }}”</p>
        </section>

        <!-- Bukti bayar -->
        <section>
          <h3 class="mb-2 text-xs font-medium tracking-wider text-muted uppercase">Bukti bayar</h3>
          <div v-if="!order.proof_url" class="flex flex-col items-center gap-1 rounded-lg border border-dashed border-line py-8 text-center text-xs text-muted">
            <AppIcon name="image" :size="20" />
            Customer belum mengunggah bukti bayar.
          </div>
          <div v-else-if="proofFailed" class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-danger/40 py-8 text-center text-xs text-danger">
            <AppIcon name="alert" :size="18" /> Gagal memuat gambar bukti.
            <a :href="order.proof_url" target="_blank" rel="noopener" class="btn btn-secondary btn-sm">Buka di tab baru</a>
          </div>
          <button
            v-else
            type="button"
            class="group relative block w-full overflow-hidden rounded-lg border border-line bg-canvas"
            aria-label="Perbesar bukti bayar"
            @click="zoomSrc = order.proof_url"
          >
            <img :src="order.proof_url" alt="Bukti bayar" class="mx-auto max-h-[420px] w-auto object-contain" @error="proofFailed = true" />
            <span class="absolute right-2 bottom-2 inline-flex items-center gap-1 rounded-md bg-black/70 px-2 py-1 text-[11px] text-ink opacity-80 group-hover:opacity-100">
              <AppIcon name="zoom" :size="12" /> Klik untuk perbesar
            </span>
          </button>
        </section>
      </div>
    </UiState>

    <template v-if="order && canReview" #footer>
      <button type="button" class="btn btn-danger" :disabled="!!busy" @click="openReject">
        <AppIcon name="x" :size="14" /> Tolak
      </button>
      <button type="button" class="btn btn-primary" :disabled="!!busy || (!subLoaded && !error)" @click="approve">
        <UiSpinner v-if="busy === 'approve'" :size="14" />
        <AppIcon v-else name="check" :size="14" />
        Setujui
      </button>
    </template>
  </UiDrawer>

  <ImageZoom :src="zoomSrc" :alt="order ? `Bukti bayar ${order.code}` : undefined" @close="zoomSrc = null" />

  <UiDialog :open="rejectOpen" title="Tolak pembayaran" :description="order ? `${order.code} · ${order.user_name}` : undefined" :persistent="busy === 'reject'" @close="rejectOpen = false">
    <form id="reject-form" class="space-y-3" @submit.prevent="submitReject">
      <p class="text-xs text-muted">Customer akan melihat alasan ini dan dapat mengunggah ulang bukti bayar.</p>
      <div>
        <label for="reject-reason" class="label">Alasan penolakan <span class="text-danger">*</span></label>
        <textarea
          id="reject-reason"
          v-model="rejectReason"
          rows="3"
          maxlength="500"
          class="input"
          :class="rejectError && 'input-error'"
          placeholder="Contoh: nominal transfer Rp150.000, seharusnya Rp150.237"
          data-autofocus
        />
        <p v-if="rejectError" class="field-error">{{ rejectError }}</p>
      </div>
      <div class="flex flex-wrap gap-1.5">
        <button v-for="p in REJECT_PRESETS" :key="p" type="button" class="btn btn-secondary btn-sm font-normal" @click="rejectReason = p">{{ p }}</button>
      </div>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="busy === 'reject'" @click="rejectOpen = false">Batal</button>
      <button type="submit" form="reject-form" class="btn btn-danger" :disabled="busy === 'reject'">
        <UiSpinner v-if="busy === 'reject'" :size="14" /> Tolak order
      </button>
    </template>
  </UiDialog>
</template>
