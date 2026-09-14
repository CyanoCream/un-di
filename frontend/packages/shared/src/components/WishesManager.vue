<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { api, formatDate } from '../api'
import { confirm } from '../composables/useConfirm'
import { toast } from '../composables/useToast'
import type { Attendance, Wish, WishList } from '../types'
import UiBadge from './ui/UiBadge.vue'
import UiEmpty from './ui/UiEmpty.vue'
import UiIcon from './ui/UiIcon.vue'
import UiPagination from './ui/UiPagination.vue'
import UiSpinner from './ui/UiSpinner.vue'

const props = defineProps<{ invitationId: string }>()

const PER_PAGE = 20
const page = ref(1)
const data = ref<WishList | null>(null)
const loading = ref(false)
const busyId = ref<string | null>(null)
let reqSeq = 0

async function load() {
  const my = ++reqSeq
  loading.value = true
  try {
    const res = await api.get<WishList>(`/invitations/${props.invitationId}/wishes`, { page: page.value, per_page: PER_PAGE })
    if (my !== reqSeq) return
    data.value = { ...res, items: res.items ?? [], stats: res.stats ?? { hadir: 0, tidak: 0, ragu: 0, total_pax: 0 } }
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

const ATT: Record<Attendance, { label: string; tone: 'success' | 'danger' | 'warning' }> = {
  hadir: { label: 'Hadir', tone: 'success' },
  tidak: { label: 'Tidak hadir', tone: 'danger' },
  ragu: { label: 'Masih ragu', tone: 'warning' },
}

async function toggleHidden(w: Wish) {
  busyId.value = w.id
  try {
    const updated = await api.patch<Wish>(`/invitations/${props.invitationId}/wishes/${w.id}`, { is_hidden: !w.is_hidden })
    if (data.value) {
      const i = data.value.items.findIndex((x) => x.id === w.id)
      if (i >= 0) data.value.items[i] = { ...w, ...updated }
    }
    toast.success(updated.is_hidden ? 'Ucapan disembunyikan dari undangan' : 'Ucapan ditampilkan kembali')
  } catch (e) {
    toast.error(e)
  } finally {
    busyId.value = null
  }
}

async function remove(w: Wish) {
  const ok = await confirm({ title: 'Hapus ucapan?', message: `Ucapan dari ${w.name} akan dihapus permanen beserta data kehadirannya.`, danger: true })
  if (!ok) return
  busyId.value = w.id
  try {
    await api.del(`/invitations/${props.invitationId}/wishes/${w.id}`)
    toast.success('Ucapan dihapus')
    if (data.value && data.value.items.length === 1 && page.value > 1) page.value--
    else load()
  } catch (e) {
    toast.error(e)
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <div class="space-y-4">
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><span class="size-2 rounded-full bg-success" /> Hadir</p>
        <p class="mt-1 text-2xl font-semibold text-ink">{{ data?.stats.hadir ?? '–' }}</p>
      </div>
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><span class="size-2 rounded-full bg-danger" /> Tidak hadir</p>
        <p class="mt-1 text-2xl font-semibold text-ink">{{ data?.stats.tidak ?? '–' }}</p>
      </div>
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><span class="size-2 rounded-full bg-warning" /> Ragu</p>
        <p class="mt-1 text-2xl font-semibold text-ink">{{ data?.stats.ragu ?? '–' }}</p>
      </div>
      <div class="rounded-2xl border border-accent/25 bg-accent/8 p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><UiIcon name="users" :size="13" class="text-accent" /> Total orang hadir</p>
        <p class="mt-1 text-2xl font-semibold text-ink">{{ data?.stats.total_pax ?? '–' }}</p>
      </div>
    </div>

    <div class="relative overflow-hidden rounded-2xl border border-line bg-surface">
      <div v-if="loading && data" class="absolute inset-x-0 top-0 z-10 h-0.5 animate-pulse bg-accent" />
      <div v-if="!data" class="flex items-center justify-center gap-2 py-16 text-sm text-muted"><UiSpinner /> Memuat ucapan…</div>
      <UiEmpty
        v-else-if="!data.items.length"
        title="Belum ada ucapan"
        description="Ucapan dan konfirmasi kehadiran dari tamu akan muncul di sini."
        icon="heart"
      />
      <ul v-else class="divide-y divide-line">
        <li v-for="w in data.items" :key="w.id" class="flex gap-3 p-4" :class="w.is_hidden ? 'bg-surface-2/60' : ''">
          <div
            class="flex size-9 shrink-0 items-center justify-center rounded-full bg-accent/10 text-sm font-semibold text-accent uppercase"
            :class="w.is_hidden ? 'opacity-50' : ''"
          >
            {{ w.name.trim().charAt(0) || '?' }}
          </div>
          <div class="min-w-0 flex-1" :class="w.is_hidden ? 'opacity-60' : ''">
            <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
              <p class="font-medium text-ink">{{ w.name }}</p>
              <UiBadge :tone="ATT[w.attendance]?.tone ?? 'neutral'">
                {{ ATT[w.attendance]?.label ?? w.attendance }}<template v-if="w.attendance === 'hadir' && w.pax"> · {{ w.pax }} orang</template>
              </UiBadge>
              <UiBadge v-if="w.is_hidden"><UiIcon name="eye-off" :size="12" /> Disembunyikan</UiBadge>
              <UiBadge v-if="w.guest_id" tone="accent">Tamu terdaftar</UiBadge>
            </div>
            <p v-if="w.message" class="mt-1.5 text-sm leading-relaxed break-words whitespace-pre-line text-ink">{{ w.message }}</p>
            <p class="mt-1.5 flex items-center gap-1 text-xs text-muted"><UiIcon name="clock" :size="12" /> {{ formatDate(w.created_at, true) }}</p>
          </div>
          <div class="flex shrink-0 flex-col gap-1 sm:flex-row sm:items-start">
            <button
              type="button"
              class="rounded-lg p-2 text-muted hover:bg-surface-2 hover:text-ink disabled:opacity-40"
              :title="w.is_hidden ? 'Tampilkan' : 'Sembunyikan'"
              :aria-label="w.is_hidden ? 'Tampilkan' : 'Sembunyikan'"
              :disabled="busyId === w.id"
              @click="toggleHidden(w)"
            >
              <UiIcon :name="w.is_hidden ? 'eye' : 'eye-off'" :size="16" />
            </button>
            <button
              type="button"
              class="rounded-lg p-2 text-muted hover:bg-danger/10 hover:text-danger disabled:opacity-40"
              title="Hapus"
              aria-label="Hapus"
              :disabled="busyId === w.id"
              @click="remove(w)"
            >
              <UiIcon name="trash" :size="16" />
            </button>
          </div>
        </li>
      </ul>
      <div v-if="data && data.total > PER_PAGE" class="border-t border-line px-4 py-3">
        <UiPagination v-model:page="page" :total="data.total" :per-page="PER_PAGE" />
      </div>
    </div>
  </div>
</template>
