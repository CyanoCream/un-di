<script setup lang="ts">
import { api, errorMessage, formatDate, formatRupiah, type Order, type Paginated } from '@undangan/shared'
import UiBadge from '@undangan/shared/components/ui/UiBadge.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiEmpty from '@undangan/shared/components/ui/UiEmpty.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiPagination from '@undangan/shared/components/ui/UiPagination.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { onMounted, ref, watch } from 'vue'
import PageHeader from '../components/PageHeader.vue'
import { ORDER_STATUS } from '../lib/labels'

const PER_PAGE = 20
const page = ref(1)
const data = ref<Paginated<Order> | null>(null)
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Paginated<Order>>('/me/orders', { page: page.value, per_page: PER_PAGE })
    data.value = { ...res, items: res.items ?? [] }
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

watch(page, load)
onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-3xl px-4 pt-2 md:px-6 md:pt-10">
    <PageHeader title="Pembayaran" subtitle="Riwayat order paket Anda.">
      <UiButton size="sm" href="/paket" @click.prevent="$router.push('/paket')"><UiIcon name="plus" :size="15" /> Beli paket</UiButton>
    </PageHeader>

    <div v-if="loading && !data" class="flex items-center justify-center gap-2 py-24 text-sm text-muted"><UiSpinner /> Memuat…</div>
    <div v-else-if="error" class="card p-8 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <UiButton variant="secondary" size="sm" class="mt-3" @click="load"><UiIcon name="refresh" :size="14" /> Coba lagi</UiButton>
    </div>
    <div v-else-if="data && !data.items.length" class="card">
      <UiEmpty title="Belum ada pembayaran" description="Order paket yang Anda buat akan tampil di sini." icon="card">
        <UiButton href="/paket" @click.prevent="$router.push('/paket')">Pilih paket</UiButton>
      </UiEmpty>
    </div>
    <div v-else-if="data" class="card overflow-hidden">
      <ul class="divide-y divide-line">
        <li v-for="o in data.items" :key="o.id">
          <RouterLink :to="`/pembayaran/${o.id}`" class="flex items-center gap-3 px-4 py-4 transition hover:bg-surface-2/60 sm:px-5">
            <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent-soft text-accent">
              <UiIcon name="card" :size="18" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span class="font-medium text-ink">Paket {{ o.plan_name }}</span>
                <UiBadge :tone="ORDER_STATUS[o.status]?.tone ?? 'neutral'">{{ ORDER_STATUS[o.status]?.label ?? o.status }}</UiBadge>
              </span>
              <span class="mt-0.5 block truncate text-xs text-muted">{{ o.code }} · {{ formatDate(o.created_at, true) }}</span>
            </span>
            <span class="text-right text-sm font-semibold whitespace-nowrap text-ink">{{ formatRupiah(o.amount) }}</span>
            <UiIcon name="chevron-right" :size="16" class="text-muted" />
          </RouterLink>
        </li>
      </ul>
      <div v-if="data.total > PER_PAGE" class="border-t border-line px-5 py-3">
        <UiPagination v-model:page="page" :total="data.total" :per-page="PER_PAGE" />
      </div>
    </div>
  </div>
</template>
