<script setup lang="ts">
import { api, formatDate, type Paginated, type Subscription, type SubscriptionStatus } from '@undangan/shared'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import UiBadge from '../components/UiBadge.vue'
import UiPagination from '../components/UiPagination.vue'
import UiState from '../components/UiState.vue'
import UiTabs from '../components/UiTabs.vue'
import { SUBSCRIPTION_STATUS } from '../lib/labels'
import { usePaged } from '../lib/usePaged'
import { daysUntil, formatNumber, useDebounced } from '../lib/util'

const STATUSES: SubscriptionStatus[] = ['active', 'grace', 'expired']
const route = useRoute()
const router = useRouter()

const status = ref<SubscriptionStatus>(STATUSES.includes(route.query.status as SubscriptionStatus) ? (route.query.status as SubscriptionStatus) : 'active')
const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const qDebounced = useDebounced(q, 350)

const { items, total, page, perPage, loading, error, load } = usePaged<Subscription>('/admin/subscriptions', () => ({
  status: status.value,
  q: qDebounced.value.trim(),
}))

const counts = reactive<Partial<Record<SubscriptionStatus, number>>>({})
onMounted(() =>
  STATUSES.forEach(async (s) => {
    try {
      counts[s] = (await api.get<Paginated<Subscription>>('/admin/subscriptions', { status: s, per_page: 1 })).total
    } catch {
      /* abaikan */
    }
  }),
)
const tabs = computed(() => STATUSES.map((s) => ({ value: s, label: SUBSCRIPTION_STATUS[s].label, count: counts[s] ?? null })))

watch([status, qDebounced], () => {
  const query: Record<string, string> = {}
  if (status.value !== 'active') query.status = status.value
  if (qDebounced.value.trim()) query.q = qDebounced.value.trim()
  router.replace({ query })
})

function remaining(s: Subscription) {
  if (s.status === 'grace') return { days: daysUntil(s.grace_ends_at), label: 'tenggang' }
  if (s.status === 'active') return { days: daysUntil(s.ends_at), label: 'hari' }
  return { days: daysUntil(s.grace_ends_at), label: '' }
}
</script>

<template>
  <div>
    <PageHeader title="Langganan" subtitle="Langganan aktif, masa tenggang, dan yang sudah berakhir.">
      <template #actions>
        <button type="button" class="btn btn-secondary" @click="load"><AppIcon name="refresh" :size="14" /> Muat ulang</button>
      </template>
    </PageHeader>

    <div class="card">
      <div class="px-3 pt-1">
        <UiTabs v-model="status" :tabs="tabs" aria-label="Status langganan" />
      </div>
      <div class="flex flex-wrap items-center gap-2 border-b border-line px-3 py-2.5">
        <div class="relative w-full sm:w-72">
          <AppIcon name="search" class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted" />
          <input v-model="q" type="search" class="input h-8 pl-8" placeholder="Cari nama atau email customer…" aria-label="Cari langganan" />
        </div>
      </div>

      <UiState :loading="loading" :error="error" :empty="!items.length" empty-title="Tidak ada langganan" empty-icon="card" @retry="load">
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>Customer</th>
                <th>Paket</th>
                <th>Status</th>
                <th>Mulai</th>
                <th>Berakhir</th>
                <th>Tenggang s.d.</th>
                <th class="text-right">Sisa</th>
                <th class="text-right">Kuota</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in items" :key="s.id" class="row-link" @click="router.push(`/users/${s.user_id}`)">
                <td class="max-w-64">
                  <RouterLink :to="`/users/${s.user_id}`" class="block truncate font-medium hover:text-accent" @click.stop>{{ s.user_name }}</RouterLink>
                  <p class="truncate text-xs text-muted">{{ s.user_email }}</p>
                </td>
                <td>{{ s.plan_name }}</td>
                <td><UiBadge :tone="SUBSCRIPTION_STATUS[s.status]?.tone">{{ SUBSCRIPTION_STATUS[s.status]?.label ?? s.status }}</UiBadge></td>
                <td class="num text-muted">{{ formatDate(s.starts_at) }}</td>
                <td class="num">{{ formatDate(s.ends_at) }}</td>
                <td class="num text-muted">{{ formatDate(s.grace_ends_at) }}</td>
                <td class="num text-right">
                  <template v-if="s.status === 'active' || s.status === 'grace'">
                    <span
                      class="font-semibold"
                      :class="(remaining(s).days ?? 0) <= 3 ? 'text-danger' : (remaining(s).days ?? 0) <= 7 ? 'text-warning' : 'text-ink'"
                    >
                      {{ Math.max(0, remaining(s).days ?? 0) }}
                    </span>
                    <span class="ml-1 text-xs text-muted">{{ remaining(s).label }}</span>
                  </template>
                  <span v-else class="text-xs text-muted">berakhir</span>
                </td>
                <td class="num text-right text-xs text-muted">{{ s.max_invitations }} inv · {{ formatNumber(s.max_guests) }} tamu</td>
              </tr>
            </tbody>
          </table>
        </div>
        <UiPagination v-model:page="page" :total="total" :per-page="perPage" />
      </UiState>
    </div>
  </div>
</template>
