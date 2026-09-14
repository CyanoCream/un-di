<script setup lang="ts">
import { api, errorMessage, formatRupiah, type Order, type Paginated } from '@undangan/shared'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { computed, onMounted, ref } from 'vue'
import CreateInvitationDialog from '../components/CreateInvitationDialog.vue'
import InvitationCard from '../components/InvitationCard.vue'
import SubscriptionCard from '../components/SubscriptionCard.vue'
import { useAccountState } from '../lib/account'
import { auth, firstName } from '../lib/auth'
import { ORDER_STATUS, greeting } from '../lib/labels'

const acc = useAccountState()
const pendingOrders = ref<Order[]>([])
const createOpen = ref(false)

async function loadOrders() {
  try {
    const res = await api.get<Paginated<Order>>('/me/orders', { per_page: 10 })
    pendingOrders.value = (res.items ?? []).filter((o) => o.status === 'awaiting_payment' || o.status === 'awaiting_confirmation')
  } catch {
    pendingOrders.value = []
  }
}

onMounted(() => {
  acc.load()
  loadOrders()
})

const name = computed(() => firstName(auth.user))
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 pt-2 md:px-6 md:pt-10">
    <div class="mb-6">
      <p class="text-sm text-muted">{{ greeting() }},</p>
      <h1 class="heading text-4xl sm:text-5xl">{{ name || 'Selamat datang' }}</h1>
    </div>

    <!-- Order menunggu -->
    <div v-if="pendingOrders.length" class="mb-6 space-y-3">
      <RouterLink
        v-for="o in pendingOrders"
        :key="o.id"
        :to="`/pembayaran/${o.id}`"
        class="flex items-center gap-3 rounded-2xl border p-4 transition hover:shadow-soft"
        :class="o.status === 'awaiting_payment' ? 'border-warning/30 bg-warning/10' : 'border-accent/25 bg-accent/8'"
      >
        <span
          class="flex size-10 shrink-0 items-center justify-center rounded-full"
          :class="o.status === 'awaiting_payment' ? 'bg-warning/15 text-warning' : 'bg-accent/15 text-accent'"
        >
          <UiIcon :name="o.status === 'awaiting_payment' ? 'card' : 'clock'" :size="18" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-semibold text-ink">
            {{ o.status === 'awaiting_payment' ? 'Selesaikan pembayaran' : 'Pembayaran sedang diverifikasi' }}
          </span>
          <span class="block truncate text-xs text-muted">
            Paket {{ o.plan_name }} · {{ formatRupiah(o.amount) }} · {{ ORDER_STATUS[o.status].label }}
          </span>
        </span>
        <UiIcon name="chevron-right" :size="18" class="text-muted" />
      </RouterLink>
    </div>

    <div v-if="acc.loading.value" class="flex items-center justify-center gap-2 py-24 text-sm text-muted"><UiSpinner /> Memuat…</div>
    <div v-else-if="acc.error.value" class="card p-8 text-center">
      <p class="text-sm text-danger">{{ errorMessage(acc.error.value) }}</p>
      <UiButton variant="secondary" size="sm" class="mt-3" @click="acc.load()"><UiIcon name="refresh" :size="14" /> Coba lagi</UiButton>
    </div>

    <div v-else class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
      <section class="order-2 lg:order-1">
        <div class="mb-3 flex items-center justify-between gap-3">
          <h2 class="heading text-2xl">Undangan saya</h2>
          <RouterLink v-if="acc.invitations.value.length" to="/undangan" class="text-sm font-medium text-accent hover:underline">Lihat semua</RouterLink>
        </div>

        <div v-if="acc.invitations.value.length" class="grid gap-4 sm:grid-cols-2">
          <InvitationCard v-for="inv in acc.invitations.value.slice(0, 4)" :key="inv.id" :invitation="inv" />
        </div>

        <div v-else class="card flex flex-col items-center px-6 py-10 text-center">
          <svg class="mb-3 w-24 text-accent/50" viewBox="0 0 120 80" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <rect x="15" y="20" width="90" height="55" rx="6" />
            <path d="m15 26 45 30 45-30" />
            <path d="M60 18c-4-7-14-6-14 2 0 6 14 13 14 13s14-7 14-13c0-8-10-9-14-2Z" fill="currentColor" fill-opacity=".15" />
          </svg>
          <p class="heading text-2xl">Belum ada undangan</p>
          <p class="mt-1 max-w-sm text-sm text-muted">
            {{ acc.createBlockReason.value ?? 'Buat undangan pertama Anda — isi data mempelai, pilih tema, lalu bagikan ke tamu.' }}
          </p>
          <UiButton v-if="!acc.createBlockReason.value" class="mt-4" @click="createOpen = true"><UiIcon name="plus" :size="16" /> Buat undangan</UiButton>
          <UiButton v-else-if="!acc.subscription.value || acc.subscription.value.status !== 'active'" class="mt-4" href="/paket" @click.prevent="$router.push('/paket')">
            Pilih paket
          </UiButton>
        </div>
      </section>

      <aside class="order-1 space-y-4 lg:order-2">
        <SubscriptionCard :subscription="acc.subscription.value" :invitation-count="acc.invitationTotal.value" />
        <div class="card hidden p-5 lg:block">
          <p class="mb-3 text-sm font-semibold text-ink">Langkah membuat undangan</p>
          <ol class="space-y-3 text-sm">
            <li v-for="(s, i) in ['Pilih paket & bayar via QRIS', 'Isi data mempelai dan acara', 'Pilih tema favorit', 'Atur subdomain & publikasikan', 'Tambahkan tamu dan kirim via WhatsApp']" :key="s" class="flex gap-3">
              <span class="flex size-6 shrink-0 items-center justify-center rounded-full bg-accent-soft text-xs font-semibold text-accent">{{ i + 1 }}</span>
              <span class="pt-0.5 text-muted">{{ s }}</span>
            </li>
          </ol>
        </div>
      </aside>
    </div>

    <CreateInvitationDialog v-model:open="createOpen" />
  </div>
</template>
