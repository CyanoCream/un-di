<script setup lang="ts">
import { api, formatDate, type Order, type OrderStatus, type Paginated, type Plan } from '@undangan/shared'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AmountText from '../components/AmountText.vue'
import AppIcon from '../components/AppIcon.vue'
import OrderDrawer from '../components/OrderDrawer.vue'
import PageHeader from '../components/PageHeader.vue'
import UiBadge from '../components/UiBadge.vue'
import UiPagination from '../components/UiPagination.vue'
import UiState from '../components/UiState.vue'
import UiTabs from '../components/UiTabs.vue'
import { ORDER_STATUS } from '../lib/labels'
import { usePaged } from '../lib/usePaged'
import { itemsOf, relativeTime, useDebounced } from '../lib/util'

const STATUSES: OrderStatus[] = ['awaiting_confirmation', 'awaiting_payment', 'paid', 'rejected', 'expired']

const route = useRoute()
const router = useRouter()

const status = ref<OrderStatus>(STATUSES.includes(route.query.status as OrderStatus) ? (route.query.status as OrderStatus) : 'awaiting_confirmation')
const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const qDebounced = useDebounced(q, 350)
const openId = ref<string | null>(typeof route.query.open === 'string' ? route.query.open : null)

const { items, total, page, perPage, loading, error, load } = usePaged<Order>('/admin/orders', () => ({ status: status.value, q: qDebounced.value.trim() }))

const counts = reactive<Partial<Record<OrderStatus, number>>>({})
async function loadCounts() {
  await Promise.all(
    STATUSES.map(async (s) => {
      try {
        const res = await api.get<Paginated<Order>>('/admin/orders', { status: s, per_page: 1 })
        counts[s] = res.total
      } catch {
        /* abaikan */
      }
    }),
  )
}

const plans = ref<Plan[]>([])
onMounted(async () => {
  loadCounts()
  try {
    plans.value = itemsOf(await api.get<{ items: Plan[] }>('/admin/plans'))
  } catch {
    /* durasi paket hanya informasi di konfirmasi */
  }
})

const tabs = computed(() => STATUSES.map((s) => ({ value: s, label: ORDER_STATUS[s].label, count: counts[s] ?? null })))

watch(
  () => route.query.open,
  (v) => {
    if (typeof v === 'string' && v !== openId.value) openId.value = v
  },
)

watch([status, qDebounced, openId], () => {
  const query: Record<string, string> = {}
  if (status.value !== 'awaiting_confirmation') query.status = status.value
  if (qDebounced.value.trim()) query.q = qDebounced.value.trim()
  if (openId.value) query.open = openId.value
  router.replace({ query })
})

function onUpdated(o: Order) {
  const i = items.value.findIndex((x) => x.id === o.id)
  if (i >= 0) items.value[i] = o
  loadCounts()
}

function refresh() {
  load()
  loadCounts()
}
</script>

<template>
  <div>
    <PageHeader title="Order Pembayaran" subtitle="Cocokkan nominal persis (dengan kode unik) dengan mutasi QRIS, lalu setujui atau tolak.">
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="refresh"><AppIcon name="refresh" :size="14" /> Muat ulang</button>
      </template>
    </PageHeader>

    <div class="card">
      <div class="px-3 pt-1">
        <UiTabs v-model="status" :tabs="tabs" aria-label="Status order" />
      </div>
      <div class="flex flex-wrap items-center gap-2 border-b border-line px-3 py-2.5">
        <div class="relative w-full sm:w-72">
          <AppIcon name="search" class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted" />
          <input v-model="q" type="search" class="input h-8 pl-8" placeholder="Cari kode order, nama, email…" aria-label="Cari order" />
        </div>
      </div>

      <UiState
        :loading="loading"
        :error="error"
        :empty="!items.length"
        :empty-title="q ? 'Tidak ada order yang cocok' : `Tidak ada order ${ORDER_STATUS[status].label.toLowerCase()}`"
        empty-icon="orders"
        @retry="load"
      >
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>Kode</th>
                <th>Customer</th>
                <th>Paket</th>
                <th class="text-right">Nominal</th>
                <th>Dibuat</th>
                <th>Status</th>
                <th class="w-10"><span class="sr-only">Bukti</span></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="o in items"
                :key="o.id"
                class="row-link"
                :class="openId === o.id && 'bg-accent/5'"
                tabindex="0"
                @click="openId = o.id"
                @keydown.enter="openId = o.id"
              >
                <td class="font-mono text-xs">{{ o.code }}</td>
                <td class="max-w-60">
                  <p class="truncate font-medium">{{ o.user_name }}</p>
                  <p class="truncate text-xs text-muted">{{ o.user_email }}</p>
                </td>
                <td class="text-muted">{{ o.plan_name }}</td>
                <td class="text-right font-medium"><AmountText :amount="o.amount" :unique-code="o.unique_code" /></td>
                <td>
                  <p class="num">{{ formatDate(o.created_at, true) }}</p>
                  <p class="text-xs text-muted">{{ relativeTime(o.created_at) }}</p>
                </td>
                <td><UiBadge :tone="ORDER_STATUS[o.status]?.tone">{{ ORDER_STATUS[o.status]?.label ?? o.status }}</UiBadge></td>
                <td class="text-muted">
                  <AppIcon v-if="o.proof_url" name="image" :size="15" title="Ada bukti bayar" />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <UiPagination v-model:page="page" :total="total" :per-page="perPage" />
      </UiState>
    </div>

    <OrderDrawer :order-id="openId" :plans="plans" @close="openId = null" @updated="onUpdated" />
  </div>
</template>
