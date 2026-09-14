<script setup lang="ts">
import { api, formatDate, formatRupiah, type AdminStats, type InvitationSummary, type Order, type Paginated } from '@undangan/shared'
import { computed, onMounted, ref } from 'vue'
import AppIcon, { type IconName } from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import UiBadge from '../components/UiBadge.vue'
import UiState from '../components/UiState.vue'
import { auth } from '../lib/auth'
import { INVITATION_STATUS } from '../lib/labels'
import { errMsg, formatNumber, relativeTime } from '../lib/util'

const stats = ref<AdminStats | null>(null)
const statsError = ref('')
const statsLoading = ref(true)

const orders = ref<Order[]>([])
const ordersTotal = ref(0)
const ordersError = ref('')
const ordersLoading = ref(true)

const invitations = ref<InvitationSummary[]>([])
const invError = ref('')
const invLoading = ref(true)

async function loadStats() {
  statsLoading.value = true
  statsError.value = ''
  try {
    stats.value = await api.get<AdminStats>('/admin/stats')
  } catch (e) {
    statsError.value = errMsg(e)
  } finally {
    statsLoading.value = false
  }
}
async function loadOrders() {
  ordersLoading.value = true
  ordersError.value = ''
  try {
    const res = await api.get<Paginated<Order>>('/admin/orders', { status: 'awaiting_confirmation', per_page: 6 })
    orders.value = res.items ?? []
    ordersTotal.value = res.total ?? 0
  } catch (e) {
    ordersError.value = errMsg(e)
  } finally {
    ordersLoading.value = false
  }
}
async function loadInvitations() {
  invLoading.value = true
  invError.value = ''
  try {
    const res = await api.get<Paginated<InvitationSummary>>('/invitations', { per_page: 8 })
    invitations.value = res.items ?? []
  } catch (e) {
    invError.value = errMsg(e)
  } finally {
    invLoading.value = false
  }
}

function reloadAll() {
  loadStats()
  loadOrders()
  loadInvitations()
}
onMounted(reloadAll)

interface Tile {
  label: string
  value: string
  sub?: string
  icon: IconName
  to: string
  highlight?: boolean
}

const tiles = computed<Tile[]>(() => {
  const s = stats.value
  if (!s) return []
  const pct = s.invitations ? Math.round((s.published_invitations / s.invitations) * 100) : 0
  return [
    { label: 'Pengguna', value: formatNumber(s.users), icon: 'users', to: '/users' },
    {
      label: 'Langganan aktif',
      value: formatNumber(s.active_subscriptions),
      sub: `${formatNumber(s.grace_subscriptions)} masa tenggang`,
      icon: 'card',
      to: '/subscriptions',
    },
    {
      label: 'Order menunggu',
      value: formatNumber(s.pending_orders),
      sub: s.pending_orders ? 'Perlu dikonfirmasi' : 'Semua beres',
      icon: 'orders',
      to: '/orders',
      highlight: s.pending_orders > 0,
    },
    { label: 'Pendapatan bulan ini', value: formatRupiah(s.revenue_this_month), icon: 'wallet', to: '/orders' },
    {
      label: 'Undangan terbit',
      value: `${formatNumber(s.published_invitations)} / ${formatNumber(s.invitations)}`,
      sub: `${pct}% dari total undangan`,
      icon: 'mail',
      to: '/invitations',
    },
    { label: 'Total tamu', value: formatNumber(s.guests), icon: 'users', to: '/invitations' },
  ]
})

const greeting = computed(() => {
  const h = new Date().getHours()
  return h < 11 ? 'Selamat pagi' : h < 15 ? 'Selamat siang' : h < 19 ? 'Selamat sore' : 'Selamat malam'
})
</script>

<template>
  <div>
    <PageHeader title="Dashboard" :subtitle="`${greeting}, ${auth.user?.name ?? 'admin'}. Ringkasan operasional hari ini.`">
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="reloadAll"><AppIcon name="refresh" :size="14" /> Muat ulang</button>
      </template>
    </PageHeader>

    <!-- Stat tiles -->
    <div v-if="statsError" class="card mb-5">
      <UiState :error="statsError" compact @retry="loadStats" />
    </div>
    <div v-else class="mb-5 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
      <template v-if="statsLoading && !stats">
        <div v-for="i in 6" :key="i" class="card h-[92px] animate-pulse p-4">
          <div class="h-3 w-20 rounded bg-surface-2" />
          <div class="mt-3 h-6 w-16 rounded bg-surface-2" />
        </div>
      </template>
      <RouterLink
        v-for="t in tiles"
        :key="t.label"
        :to="t.to"
        class="card group relative overflow-hidden p-4 transition-colors hover:border-accent/40"
        :class="t.highlight && 'border-warning/40'"
      >
        <div class="flex items-center justify-between gap-2">
          <p class="truncate text-xs text-muted">{{ t.label }}</p>
          <AppIcon :name="t.icon" :size="14" :class="t.highlight ? 'text-warning' : 'text-muted/60 group-hover:text-accent'" />
        </div>
        <p class="num mt-2 truncate text-xl font-semibold tracking-tight" :class="t.highlight && 'text-warning'">{{ t.value }}</p>
        <p v-if="t.sub" class="mt-0.5 truncate text-[11px] text-muted">{{ t.sub }}</p>
      </RouterLink>
    </div>

    <div class="grid gap-5 xl:grid-cols-5">
      <!-- Order menunggu konfirmasi -->
      <section class="card xl:col-span-2" aria-labelledby="pending-title">
        <div class="card-header">
          <h2 id="pending-title" class="card-title flex items-center gap-2">
            Menunggu konfirmasi
            <span v-if="ordersTotal" class="num rounded-full bg-warning/15 px-1.5 text-[11px] text-warning">{{ ordersTotal }}</span>
          </h2>
          <RouterLink to="/orders" class="text-xs text-accent hover:underline">Lihat semua</RouterLink>
        </div>
        <UiState
          :loading="ordersLoading"
          :error="ordersError"
          :empty="!orders.length"
          empty-title="Tidak ada order menunggu"
          empty-text="Bukti bayar baru dari customer akan muncul di sini."
          empty-icon="check-circle"
          compact
          @retry="loadOrders"
        >
          <ul class="divide-y divide-line/60">
            <li v-for="o in orders" :key="o.id">
              <RouterLink :to="{ path: '/orders', query: { open: o.id } }" class="flex items-center gap-3 px-4 py-2.5 transition-colors hover:bg-surface-2/70">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-[13px] font-medium">{{ o.user_name }}</p>
                  <p class="truncate text-xs text-muted">
                    <span class="font-mono">{{ o.code }}</span> · {{ o.plan_name }}
                  </p>
                </div>
                <div class="text-right">
                  <p class="num text-[13px] font-semibold">{{ formatRupiah(o.amount) }}</p>
                  <p class="text-[11px] text-muted">{{ relativeTime(o.proof_uploaded_at ?? o.created_at) }}</p>
                </div>
                <AppIcon name="chevron-right" :size="14" class="text-muted" />
              </RouterLink>
            </li>
          </ul>
        </UiState>
      </section>

      <!-- Undangan terbaru -->
      <section class="card xl:col-span-3" aria-labelledby="recent-inv-title">
        <div class="card-header">
          <h2 id="recent-inv-title" class="card-title">Undangan terbaru</h2>
          <RouterLink to="/invitations" class="text-xs text-accent hover:underline">Lihat semua</RouterLink>
        </div>
        <UiState
          :loading="invLoading"
          :error="invError"
          :empty="!invitations.length"
          empty-title="Belum ada undangan"
          empty-icon="mail"
          compact
          @retry="loadInvitations"
        >
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>Undangan</th>
                  <th>Pemilik</th>
                  <th>Status</th>
                  <th>Acara</th>
                  <th class="text-right">Tamu</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="inv in invitations" :key="inv.id" class="row-link" @click="$router.push(`/invitations/${inv.id}`)">
                  <td class="max-w-56 truncate font-medium">
                    <RouterLink :to="`/invitations/${inv.id}`" class="hover:text-accent" @click.stop>{{ inv.title }}</RouterLink>
                  </td>
                  <td class="max-w-44 truncate text-muted">{{ inv.user_name }}</td>
                  <td><UiBadge :tone="INVITATION_STATUS[inv.status]?.tone">{{ INVITATION_STATUS[inv.status]?.label ?? inv.status }}</UiBadge></td>
                  <td class="num text-muted">{{ formatDate(inv.event_date) }}</td>
                  <td class="num text-right">{{ formatNumber(inv.guest_count) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </UiState>
      </section>
    </div>
  </div>
</template>
