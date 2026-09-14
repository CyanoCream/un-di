<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api, formatDate, formatRupiah } from '../api'
import { confirm } from '../composables/useConfirm'
import { toast } from '../composables/useToast'
import type { GiftConfirmation, GiftList, GiftType } from '../types'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiDialog from './ui/UiDialog.vue'
import UiEmpty from './ui/UiEmpty.vue'
import UiIcon from './ui/UiIcon.vue'
import UiPagination from './ui/UiPagination.vue'
import UiSpinner from './ui/UiSpinner.vue'
import UiSwitch from './ui/UiSwitch.vue'

const props = defineProps<{ invitationId: string }>()

const PER_PAGE = 20
const base = computed(() => `/invitations/${props.invitationId}/gifts`)
const page = ref(1)
const data = ref<GiftList | null>(null)
const loading = ref(false)
const busyId = ref<string | null>(null)
let reqSeq = 0

const EMPTY_SUMMARY: GiftList['summary'] = { count: 0, verified_count: 0, total_amount: 0, verified_amount: 0 }

async function load() {
  const my = ++reqSeq
  loading.value = true
  try {
    const res = await api.get<GiftList>(base.value, { page: page.value, per_page: PER_PAGE })
    if (my !== reqSeq) return
    data.value = { ...res, items: res.items ?? [], summary: res.summary ?? EMPTY_SUMMARY }
  } catch (e) {
    if (my === reqSeq) toast.error(e)
  } finally {
    if (my === reqSeq) loading.value = false
  }
}

watch(page, load)
watch(
  () => props.invitationId,
  () => {
    page.value = 1
    load()
  },
)
onMounted(load)

const summary = computed(() => data.value?.summary ?? EMPTY_SUMMARY)
const exportUrl = computed(() => api.url(`${base.value}/export.xlsx`))

const TYPE: Record<GiftType, { label: string; icon: 'card' | 'gift' }> = {
  transfer: { label: 'Transfer', icon: 'card' },
  kado: { label: 'Kado', icon: 'gift' },
}

async function toggleVerified(g: GiftConfirmation, value: boolean) {
  busyId.value = g.id
  const prev = g.is_verified
  g.is_verified = value // optimistik
  try {
    const updated = await api.patch<GiftConfirmation>(`${base.value}/${g.id}`, { is_verified: value })
    if (data.value) {
      const i = data.value.items.findIndex((x) => x.id === g.id)
      if (i >= 0) data.value.items[i] = { ...g, ...updated }
      const s = data.value.summary
      if (prev !== updated.is_verified) {
        const d = updated.is_verified ? 1 : -1
        s.verified_count = Math.max(0, s.verified_count + d)
        s.verified_amount = Math.max(0, s.verified_amount + d * (g.amount ?? 0))
      }
    }
    toast.success(updated.is_verified ? 'Ditandai sudah diterima' : 'Tanda diterima dihapus')
  } catch (e) {
    g.is_verified = prev
    toast.error(e)
  } finally {
    busyId.value = null
  }
}

async function remove(g: GiftConfirmation) {
  const ok = await confirm({ title: 'Hapus konfirmasi hadiah?', message: `Konfirmasi dari ${g.name} beserta foto buktinya akan dihapus permanen.`, danger: true })
  if (!ok) return
  busyId.value = g.id
  try {
    await api.del(`${base.value}/${g.id}`)
    toast.success('Konfirmasi hadiah dihapus')
    if (data.value && data.value.items.length === 1 && page.value > 1) page.value--
    else load()
  } catch (e) {
    toast.error(e)
  } finally {
    busyId.value = null
  }
}

// ---------- Zoom bukti ----------
const zoom = ref<GiftConfirmation | null>(null)
const zoomOpen = computed({
  get: () => !!zoom.value,
  set: (v: boolean) => {
    if (!v) zoom.value = null
  },
})
const brokenIds = ref(new Set<string>())
</script>

<template>
  <div class="space-y-4">
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-3">
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><UiIcon name="gift" :size="13" /> Konfirmasi</p>
        <p class="mt-1 text-2xl font-semibold text-ink tabular-nums">{{ data ? summary.count : '–' }}</p>
      </div>
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><UiIcon name="check-circle" :size="13" class="text-success" /> Terverifikasi</p>
        <p class="mt-1 text-2xl font-semibold text-ink tabular-nums">
          {{ data ? summary.verified_count : '–' }}<span v-if="data" class="ml-1 text-sm font-normal text-muted">/ {{ summary.count }}</span>
        </p>
        <p v-if="data && summary.verified_amount" class="truncate text-xs text-muted tabular-nums">{{ formatRupiah(summary.verified_amount) }}</p>
      </div>
      <div class="col-span-2 rounded-2xl border border-accent/25 bg-accent/8 p-4 lg:col-span-1">
        <p class="flex items-center gap-1.5 text-xs text-muted"><UiIcon name="card" :size="13" class="text-accent" /> Total nominal</p>
        <p class="mt-1 truncate text-2xl font-semibold text-ink tabular-nums">{{ data ? formatRupiah(summary.total_amount) : '–' }}</p>
        <p class="text-xs text-muted">Sesuai yang diisi tamu</p>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-between gap-2">
      <p class="text-sm text-muted">Cocokkan bukti dengan mutasi rekening, lalu tandai "Sudah diterima".</p>
      <div class="flex gap-2">
        <UiButton variant="ghost" size="sm" aria-label="Muat ulang" @click="load"><UiIcon name="refresh" :size="15" :class="loading ? 'animate-spin' : ''" /></UiButton>
        <UiButton variant="secondary" size="sm" :href="exportUrl" download><UiIcon name="download" :size="15" /> Export Excel</UiButton>
      </div>
    </div>

    <div class="relative overflow-hidden rounded-2xl border border-line bg-surface">
      <div v-if="loading && data" class="absolute inset-x-0 top-0 z-10 h-0.5 animate-pulse bg-accent" />
      <div v-if="!data" class="flex items-center justify-center gap-2 py-16 text-sm text-muted"><UiSpinner /> Memuat konfirmasi hadiah…</div>
      <UiEmpty
        v-else-if="!data.items.length"
        title="Belum ada konfirmasi hadiah"
        description="Tamu bisa mengirim foto bukti transfer atau kado dari halaman undangan. Aktifkan lewat Isi Data → Amplop Digital → “Izinkan tamu mengirim bukti transfer/kado”."
        icon="gift"
      />
      <ul v-else class="divide-y divide-line">
        <li v-for="g in data.items" :key="g.id" class="flex gap-3 p-4" :class="g.is_verified ? 'bg-success/5' : ''">
          <button
            type="button"
            class="relative size-20 shrink-0 overflow-hidden rounded-xl border border-line bg-surface-2 sm:size-24"
            :aria-label="`Lihat bukti dari ${g.name}`"
            @click="zoom = g"
          >
            <img
              v-if="!brokenIds.has(g.id)"
              :src="g.proof_url"
              alt=""
              loading="lazy"
              class="size-full object-cover"
              @error="brokenIds.add(g.id)"
            />
            <span v-else class="flex size-full items-center justify-center text-muted"><UiIcon name="image" :size="22" /></span>
            <span class="absolute right-1 bottom-1 rounded-md bg-ink/60 p-0.5 text-surface"><UiIcon name="search" :size="12" /></span>
          </button>

          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
              <p class="font-medium text-ink">{{ g.name }}</p>
              <UiBadge :tone="g.type === 'transfer' ? 'accent' : 'warning'"><UiIcon :name="TYPE[g.type]?.icon ?? 'gift'" :size="11" /> {{ TYPE[g.type]?.label ?? g.type }}</UiBadge>
              <UiBadge v-if="g.guest_name" tone="success" :title="g.guest_name">tamu terdaftar: {{ g.guest_name }}</UiBadge>
            </div>
            <p class="mt-1 text-sm text-ink">
              <strong v-if="g.amount" class="tabular-nums">{{ formatRupiah(g.amount) }}</strong>
              <span v-if="g.amount && g.account_label" class="text-muted"> · </span>
              <span v-if="g.account_label" class="text-muted">{{ g.account_label }}</span>
            </p>
            <p v-if="g.message" class="mt-1 text-sm leading-relaxed break-words whitespace-pre-line text-ink">{{ g.message }}</p>
            <p class="mt-1.5 flex items-center gap-1 text-xs text-muted"><UiIcon name="clock" :size="12" /> {{ formatDate(g.created_at, true) }}</p>
            <div class="mt-2.5 flex flex-wrap items-center justify-between gap-2">
              <UiSwitch
                :model-value="g.is_verified"
                :disabled="busyId === g.id"
                label="Sudah diterima"
                :description="g.is_verified && g.verified_at ? formatDate(g.verified_at, true) : undefined"
                @update:model-value="toggleVerified(g, $event)"
              />
              <button
                type="button"
                class="rounded-lg p-2 text-muted hover:bg-danger/10 hover:text-danger disabled:opacity-40"
                title="Hapus"
                aria-label="Hapus"
                :disabled="busyId === g.id"
                @click="remove(g)"
              >
                <UiIcon name="trash" :size="16" />
              </button>
            </div>
          </div>
        </li>
      </ul>
      <div v-if="data && data.total > PER_PAGE" class="border-t border-line px-4 py-3">
        <UiPagination v-model:page="page" :total="data.total" :per-page="PER_PAGE" />
      </div>
    </div>

    <UiDialog v-model:open="zoomOpen" :title="zoom ? `Bukti dari ${zoom.name}` : ''" size="lg">
      <template v-if="zoom" #actions>
        <a :href="zoom.proof_url" target="_blank" rel="noopener" class="-m-1.5 rounded-lg p-1.5 text-muted hover:bg-surface-2 hover:text-ink" aria-label="Buka di tab baru">
          <UiIcon name="external" :size="18" />
        </a>
      </template>
      <div v-if="zoom" class="space-y-3">
        <img :src="zoom.proof_url" :alt="`Bukti ${TYPE[zoom.type]?.label ?? ''} dari ${zoom.name}`" class="mx-auto max-h-[70dvh] w-auto rounded-xl bg-surface-2 object-contain" />
        <p class="text-center text-sm text-muted">
          {{ TYPE[zoom.type]?.label }}<template v-if="zoom.amount"> · {{ formatRupiah(zoom.amount) }}</template><template v-if="zoom.account_label"> · {{ zoom.account_label }}</template>
        </p>
      </div>
    </UiDialog>
  </div>
</template>
