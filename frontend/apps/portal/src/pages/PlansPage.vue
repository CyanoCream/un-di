<script setup lang="ts">
import { api, errorMessage, formatRupiah, toast, type Order, type Paginated, type Plan, type Subscription } from '@undangan/shared'
import UiBadge from '@undangan/shared/components/ui/UiBadge.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiEmpty from '@undangan/shared/components/ui/UiEmpty.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'

const router = useRouter()
const plans = ref<Plan[]>([])
const current = ref<Subscription | null>(null)
const pending = ref<Order | null>(null)
const loading = ref(true)
const error = ref('')
const ordering = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [p, s, o] = await Promise.all([
      api.get<{ items: Plan[] }>('/plans'),
      api.get<{ current: Subscription | null }>('/me/subscription').catch(() => ({ current: null })),
      api.get<Paginated<Order>>('/me/orders', { per_page: 5 }).catch(() => ({ items: [] as Order[] })),
    ])
    plans.value = (p.items ?? []).filter((x) => x.is_active !== false).sort((a, b) => a.sort_order - b.sort_order || a.price - b.price)
    current.value = s.current
    pending.value = (o.items ?? []).find((x) => x.status === 'awaiting_payment' || x.status === 'awaiting_confirmation') ?? null
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

onMounted(load)

const hasActive = computed(() => current.value?.status === 'active' || current.value?.status === 'grace')

function durationLabel(days: number) {
  if (days % 365 === 0) return `${days / 365} tahun`
  if (days % 30 === 0) return `${days / 30} bulan`
  return `${days} hari`
}

function features(p: Plan) {
  const f = [
    `Masa aktif ${durationLabel(p.duration_days)}`,
    p.max_invitations > 0 ? `${p.max_invitations} undangan` : 'Undangan tanpa batas',
    p.max_guests > 0 ? `Hingga ${p.max_guests.toLocaleString('id-ID')} tamu per undangan` : 'Tamu tanpa batas',
    'Link personal & kirim via WhatsApp',
    'RSVP, ucapan, dan amplop digital',
  ]
  if (p.grace_days > 0) f.push(`Masa tenggang ${p.grace_days} hari`)
  if (p.allow_custom_domain) f.push('Domain sendiri (mis. www.namakalian.com)')
  if (p.allow_checkin) f.push('Check-in QR di venue')
  return f
}

async function choose(p: Plan) {
  ordering.value = p.id
  try {
    const order = await api.post<Order>('/me/orders', { plan_id: p.id })
    router.push(`/pembayaran/${order.id}`)
  } catch (e) {
    toast.error(e)
  } finally {
    ordering.value = null
  }
}
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 pt-2 md:px-6 md:pt-10">
    <PageHeader
      :title="hasActive ? 'Perpanjang atau upgrade' : 'Pilih paket'"
      subtitle="Bayar mudah via QRIS. Paket aktif setelah pembayaran diverifikasi admin."
    >
      <UiButton variant="ghost" size="sm" href="/pembayaran" @click.prevent="$router.push('/pembayaran')">
        <UiIcon name="list" :size="15" /> Riwayat pembayaran
      </UiButton>
    </PageHeader>

    <RouterLink
      v-if="pending"
      :to="`/pembayaran/${pending.id}`"
      class="mb-6 flex items-center gap-3 rounded-2xl border border-warning/30 bg-warning/10 p-4 text-sm transition hover:shadow-soft"
    >
      <UiIcon name="clock" :size="18" class="text-warning" />
      <span class="flex-1 text-ink">
        Anda punya order <strong>{{ pending.plan_name }}</strong> ({{ formatRupiah(pending.amount) }}) yang
        {{ pending.status === 'awaiting_payment' ? 'belum dibayar' : 'sedang diverifikasi' }}.
      </span>
      <span class="font-medium text-accent">Lihat</span>
    </RouterLink>

    <div v-if="loading" class="flex items-center justify-center gap-2 py-24 text-sm text-muted"><UiSpinner /> Memuat paket…</div>
    <div v-else-if="error" class="card p-8 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <UiButton variant="secondary" size="sm" class="mt-3" @click="load"><UiIcon name="refresh" :size="14" /> Coba lagi</UiButton>
    </div>
    <div v-else-if="!plans.length" class="card"><UiEmpty title="Belum ada paket tersedia" icon="gift" /></div>

    <div v-else class="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
      <article
        v-for="p in plans"
        :key="p.id"
        class="card relative flex flex-col p-6"
        :class="current?.plan_id === p.id && hasActive ? 'ring-2 ring-accent/40' : ''"
      >
        <div class="flex items-start justify-between gap-2">
          <h2 class="heading text-3xl">{{ p.name }}</h2>
          <UiBadge v-if="current?.plan_id === p.id && hasActive" tone="accent">Paket Anda</UiBadge>
        </div>
        <p v-if="p.description" class="mt-1 text-sm text-muted">{{ p.description }}</p>
        <div class="mt-5 flex items-baseline gap-1">
          <span class="text-3xl font-semibold tracking-tight text-ink">{{ formatRupiah(p.price) }}</span>
          <span class="text-sm text-muted">/ {{ durationLabel(p.duration_days) }}</span>
        </div>
        <ul class="mt-5 flex-1 space-y-2.5 border-t border-line pt-5 text-sm">
          <li v-for="f in features(p)" :key="f" class="flex items-start gap-2.5">
            <span class="mt-0.5 flex size-4.5 shrink-0 items-center justify-center rounded-full bg-accent/12 text-accent">
              <UiIcon name="check" :size="11" :stroke-width="3" />
            </span>
            <span class="text-ink">{{ f }}</span>
          </li>
        </ul>
        <UiButton
          class="mt-6 h-11!"
          block
          :variant="current?.plan_id === p.id || !hasActive ? 'primary' : 'secondary'"
          :loading="ordering === p.id"
          :disabled="!!ordering && ordering !== p.id"
          @click="choose(p)"
        >
          {{ current?.plan_id === p.id && hasActive ? 'Perpanjang paket' : hasActive ? 'Pilih paket ini' : 'Pilih paket' }}
        </UiButton>
      </article>
    </div>
  </div>
</template>
