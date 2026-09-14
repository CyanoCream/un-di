<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api, formatDate } from '../api'
import { confirm } from '../composables/useConfirm'
import { toast } from '../composables/useToast'
import type { Attendance, AttendanceItem, AttendanceReport, Invitation } from '../types'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiDialog from './ui/UiDialog.vue'
import UiEmpty from './ui/UiEmpty.vue'
import UiIcon from './ui/UiIcon.vue'
import UiInput from './ui/UiInput.vue'
import UiPagination from './ui/UiPagination.vue'
import UiSpinner from './ui/UiSpinner.vue'
import UiStepper from './ui/UiStepper.vue'
import UiSwitch from './ui/UiSwitch.vue'

const props = defineProps<{ invitation: Invitation }>()

type StatusFilter = 'all' | 'checked_in' | 'not_checked_in'
const PER_PAGE = 25
const REFRESH_MS = 15_000

const base = computed(() => `/invitations/${props.invitation.id}`)
const q = ref('')
const status = ref<StatusFilter>('all')
const page = ref(1)
const data = ref<AttendanceReport | null>(null)
const loading = ref(false)
let reqSeq = 0

async function load(silent = false) {
  const my = ++reqSeq
  loading.value = true
  try {
    const res = await api.get<AttendanceReport>(`${base.value}/attendance`, { q: q.value.trim(), status: status.value, page: page.value, per_page: PER_PAGE })
    if (my !== reqSeq) return
    data.value = { ...res, items: res.items ?? [] }
  } catch (e) {
    if (my === reqSeq && !silent) toast.error(e)
  } finally {
    if (my === reqSeq) loading.value = false
  }
}

let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(q, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    load()
  }, 300)
})
watch(status, () => {
  page.value = 1
  load()
})
watch(page, () => load())
watch(
  () => props.invitation.id,
  () => {
    page.value = 1
    load()
  },
)
onMounted(() => load())

// ---------- Auto refresh (hari-H) ----------
const autoRefresh = ref(false)
let timer: ReturnType<typeof setInterval> | undefined
watch(autoRefresh, (on) => {
  if (timer) clearInterval(timer)
  timer = undefined
  if (on) {
    load(true)
    timer = setInterval(() => {
      if (document.visibilityState === 'visible') load(true)
    }, REFRESH_MS)
  }
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
  if (searchTimer) clearTimeout(searchTimer)
})

// ---------- Ringkasan ----------
const summary = computed(() => data.value?.summary)
const pct = computed(() => {
  const s = summary.value
  return s && s.invited_guests > 0 ? Math.min(100, Math.round((s.checked_in_guests / s.invited_guests) * 100)) : 0
})
const paxPct = computed(() => {
  const s = summary.value
  return s && s.invited_pax > 0 ? Math.min(100, Math.round((s.checked_in_pax / s.invited_pax) * 100)) : 0
})
const checkinOff = computed(() => !props.invitation.settings?.checkin_enabled)

const exportUrl = computed(() => api.url(`${base.value}/attendance/export.xlsx`, { q: q.value.trim(), status: status.value }))
const filtering = computed(() => !!q.value.trim() || status.value !== 'all')

const FILTERS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'Semua' },
  { value: 'checked_in', label: 'Sudah datang' },
  { value: 'not_checked_in', label: 'Belum datang' },
]

const RSVP: Record<Attendance, { label: string; tone: 'success' | 'danger' | 'warning' }> = {
  hadir: { label: 'Hadir', tone: 'success' },
  tidak: { label: 'Tidak hadir', tone: 'danger' },
  ragu: { label: 'Ragu', tone: 'warning' },
}

function time(iso: string | null) {
  if (!iso) return '-'
  return new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit' }).format(new Date(iso))
}

// ---------- Check-in manual & batal ----------
const dialogOpen = ref(false)
const target = ref<AttendanceItem | null>(null)
const pax = ref(1)
const busyId = ref<string | null>(null)

function openCheckin(item: AttendanceItem) {
  target.value = item
  pax.value = Math.max(1, item.pax || 1)
  dialogOpen.value = true
}

async function doCheckin() {
  const item = target.value
  if (!item) return
  busyId.value = item.guest_id
  try {
    await api.post(`${base.value}/guests/${item.guest_id}/checkin`, { pax: pax.value })
    toast.success(`${item.name} ditandai sudah datang (${pax.value} orang)`)
    dialogOpen.value = false
    load(true)
  } catch (e) {
    toast.error(e)
  } finally {
    busyId.value = null
  }
}

async function undo(item: AttendanceItem) {
  const ok = await confirm({
    title: 'Batalkan check-in?',
    message: `Status kedatangan ${item.name} akan dihapus dan tamu bisa check-in ulang.`,
    confirmText: 'Batalkan check-in',
    danger: true,
  })
  if (!ok) return
  busyId.value = item.guest_id
  try {
    await api.del(`${base.value}/guests/${item.guest_id}/checkin`)
    toast.success('Check-in dibatalkan')
    load(true)
  } catch (e) {
    toast.error(e)
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <div class="space-y-4">
    <p v-if="checkinOff" class="flex items-start gap-2 rounded-2xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-ink">
      <UiIcon name="info" :size="16" class="mt-0.5 text-warning" />
      <span>Check-in di venue belum aktif. Data RSVP tetap ditampilkan; Anda juga bisa menandai kedatangan tamu secara manual.</span>
    </p>

    <!-- Ringkasan -->
    <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><UiIcon name="users" :size="13" /> Diundang</p>
        <p class="mt-1 text-2xl font-semibold text-ink tabular-nums">{{ summary?.invited_guests ?? '–' }}<span class="ml-1 text-sm font-normal text-muted">tamu</span></p>
        <p class="text-xs text-muted tabular-nums">{{ summary?.invited_pax ?? '–' }} orang</p>
      </div>
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><span class="size-2 rounded-full bg-success" /> RSVP hadir</p>
        <p class="mt-1 text-2xl font-semibold text-ink tabular-nums">{{ summary?.rsvp_hadir ?? '–' }}<span class="ml-1 text-sm font-normal text-muted">tamu</span></p>
        <p class="text-xs text-muted tabular-nums">{{ summary?.rsvp_pax ?? '–' }} orang</p>
      </div>
      <div class="rounded-2xl border border-accent/25 bg-accent/8 p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><UiIcon name="user-check" :size="13" class="text-accent" /> Sudah datang</p>
        <p class="mt-1 text-2xl font-semibold text-ink tabular-nums">
          {{ summary?.checked_in_guests ?? '–' }}<span class="ml-1 text-sm font-normal text-muted">tamu · {{ pct }}%</span>
        </p>
        <p class="text-xs text-muted tabular-nums">{{ summary?.checked_in_pax ?? '–' }} orang</p>
      </div>
      <div class="rounded-2xl border border-line bg-surface p-4">
        <p class="flex items-center gap-1.5 text-xs text-muted"><span class="size-2 rounded-full bg-muted/60" /> Belum datang</p>
        <p class="mt-1 text-2xl font-semibold text-ink tabular-nums">{{ summary?.not_checked_in_guests ?? '–' }}<span class="ml-1 text-sm font-normal text-muted">tamu</span></p>
      </div>
    </div>

    <div class="rounded-2xl border border-line bg-surface p-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <p class="text-sm text-ink">
          Kehadiran <strong class="tabular-nums">{{ pct }}%</strong>
          <span class="text-muted tabular-nums"> · {{ summary?.checked_in_pax ?? 0 }} dari {{ summary?.invited_pax ?? 0 }} orang ({{ paxPct }}%)</span>
        </p>
        <UiSwitch v-model="autoRefresh" label="Perbarui otomatis" description="Setiap 15 detik, untuk hari-H" />
      </div>
      <div class="mt-3 h-2.5 overflow-hidden rounded-full bg-surface-2" role="progressbar" :aria-valuenow="pct" aria-valuemin="0" aria-valuemax="100">
        <div class="h-full rounded-full bg-accent transition-all duration-500" :style="{ width: `${pct}%` }" />
      </div>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="relative min-w-0 flex-1">
        <UiIcon name="search" :size="16" class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
        <UiInput v-model="q" type="search" placeholder="Cari nama, kode, atau no. HP…" class="pl-9!" />
      </div>
      <div class="flex gap-2">
        <div class="flex min-w-0 flex-1 rounded-xl border border-line bg-surface-2 p-1" role="group" aria-label="Filter kehadiran">
          <button
            v-for="f in FILTERS"
            :key="f.value"
            type="button"
            class="h-8 flex-1 rounded-lg px-2.5 text-xs font-medium whitespace-nowrap transition sm:px-3 sm:text-sm"
            :class="status === f.value ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'"
            :aria-pressed="status === f.value"
            @click="status = f.value"
          >
            {{ f.label }}
          </button>
        </div>
        <UiButton variant="secondary" :href="exportUrl" download class="shrink-0"><UiIcon name="download" :size="16" /><span class="hidden sm:inline">Export Excel</span></UiButton>
        <UiButton variant="ghost" class="shrink-0 px-3!" aria-label="Muat ulang" title="Muat ulang" @click="load()"><UiIcon name="refresh" :size="16" :class="loading ? 'animate-spin' : ''" /></UiButton>
      </div>
    </div>

    <!-- Daftar -->
    <div class="relative overflow-hidden rounded-2xl border border-line bg-surface">
      <div v-if="loading && data" class="absolute inset-x-0 top-0 z-10 h-0.5 animate-pulse bg-accent" />
      <div v-if="!data" class="flex items-center justify-center gap-2 py-16 text-sm text-muted"><UiSpinner /> Memuat kehadiran…</div>
      <UiEmpty
        v-else-if="!data.items.length"
        :title="filtering ? 'Tidak ada tamu yang cocok' : 'Belum ada tamu'"
        :description="filtering ? 'Coba kata kunci atau filter lain.' : 'Tambahkan tamu di daftar tamu agar kehadiran bisa dipantau.'"
        icon="users"
      />
      <template v-else>
        <!-- Desktop -->
        <div class="hidden overflow-x-auto md:block">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-line bg-surface-2/60 text-left text-xs font-medium tracking-wide text-muted uppercase">
                <th class="px-4 py-2.5">Nama</th>
                <th class="px-4 py-2.5">Grup</th>
                <th class="px-4 py-2.5 text-center">Diundang</th>
                <th class="px-4 py-2.5">RSVP</th>
                <th class="px-4 py-2.5">Check-in</th>
                <th class="px-4 py-2.5 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-line">
              <tr v-for="it in data.items" :key="it.guest_id" class="transition hover:bg-surface-2/50">
                <td class="px-4 py-2.5">
                  <p class="font-medium text-ink">{{ it.name }}</p>
                  <p class="font-mono text-[11px] tracking-wider text-muted">{{ it.code }}</p>
                </td>
                <td class="px-4 py-2.5 text-muted">{{ it.group_name || '-' }}</td>
                <td class="px-4 py-2.5 text-center text-ink tabular-nums">{{ it.pax }}</td>
                <td class="px-4 py-2.5">
                  <UiBadge v-if="it.rsvp_attendance" :tone="RSVP[it.rsvp_attendance]?.tone ?? 'neutral'">
                    {{ RSVP[it.rsvp_attendance]?.label ?? it.rsvp_attendance }}<template v-if="it.rsvp_attendance === 'hadir' && it.rsvp_pax"> · {{ it.rsvp_pax }}</template>
                  </UiBadge>
                  <span v-else class="text-xs text-muted">Belum RSVP</span>
                </td>
                <td class="px-4 py-2.5">
                  <template v-if="it.checked_in_at">
                    <p class="flex items-center gap-1.5 text-ink" :title="formatDate(it.checked_in_at, true)">
                      <UiIcon name="check-circle" :size="15" class="text-success" />
                      <span class="tabular-nums">{{ time(it.checked_in_at) }}</span>
                      <span class="text-muted">· {{ it.checked_in_pax ?? it.pax }} orang</span>
                    </p>
                    <p v-if="it.checked_in_by" class="text-xs text-muted">oleh {{ it.checked_in_by }}</p>
                  </template>
                  <span v-else class="text-xs text-muted">Belum datang</span>
                </td>
                <td class="px-4 py-2.5 text-right">
                  <UiButton v-if="it.checked_in_at" variant="ghost" size="sm" :disabled="busyId === it.guest_id" @click="undo(it)">
                    <UiIcon name="undo" :size="14" /> Batalkan
                  </UiButton>
                  <UiButton v-else variant="secondary" size="sm" :disabled="busyId === it.guest_id" @click="openCheckin(it)">
                    <UiIcon name="user-check" :size="14" /> Check-in
                  </UiButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Mobile -->
        <ul class="divide-y divide-line md:hidden">
          <li v-for="it in data.items" :key="it.guest_id" class="p-4">
            <div class="flex items-start gap-3">
              <span
                class="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full"
                :class="it.checked_in_at ? 'bg-success/12 text-success' : 'bg-surface-2 text-muted'"
              >
                <UiIcon :name="it.checked_in_at ? 'check' : 'clock'" :size="15" />
              </span>
              <div class="min-w-0 flex-1">
                <p class="truncate font-medium text-ink">{{ it.name }}</p>
                <p class="mt-0.5 text-xs text-muted">
                  {{ it.pax }} orang<template v-if="it.group_name"> · {{ it.group_name }}</template> · <span class="font-mono">{{ it.code }}</span>
                </p>
                <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
                  <UiBadge v-if="it.rsvp_attendance" :tone="RSVP[it.rsvp_attendance]?.tone ?? 'neutral'">
                    RSVP {{ RSVP[it.rsvp_attendance]?.label ?? it.rsvp_attendance }}<template v-if="it.rsvp_attendance === 'hadir' && it.rsvp_pax"> · {{ it.rsvp_pax }}</template>
                  </UiBadge>
                  <UiBadge v-else>Belum RSVP</UiBadge>
                  <span v-if="it.checked_in_at" class="text-xs text-ink tabular-nums">
                    Datang {{ time(it.checked_in_at) }} · {{ it.checked_in_pax ?? it.pax }} orang<template v-if="it.checked_in_by"> · {{ it.checked_in_by }}</template>
                  </span>
                </div>
              </div>
            </div>
            <div class="mt-3">
              <UiButton v-if="it.checked_in_at" variant="ghost" size="sm" block :disabled="busyId === it.guest_id" @click="undo(it)">
                <UiIcon name="undo" :size="14" /> Batalkan check-in
              </UiButton>
              <UiButton v-else variant="secondary" size="sm" block :disabled="busyId === it.guest_id" @click="openCheckin(it)">
                <UiIcon name="user-check" :size="14" /> Tandai sudah datang
              </UiButton>
            </div>
          </li>
        </ul>
      </template>
      <div v-if="data && data.total > PER_PAGE" class="border-t border-line px-4 py-3">
        <UiPagination v-model:page="page" :total="data.total" :per-page="PER_PAGE" />
      </div>
    </div>

    <UiDialog v-model:open="dialogOpen" title="Check-in manual" size="sm" :persistent="!!busyId">
      <div v-if="target" class="space-y-4 text-center">
        <div>
          <p class="text-lg font-semibold text-ink">{{ target.name }}</p>
          <p class="text-sm text-muted">
            Diundang {{ target.pax }} orang<template v-if="target.group_name"> · {{ target.group_name }}</template>
          </p>
        </div>
        <div>
          <p class="mb-2 text-sm font-medium text-ink">Jumlah yang datang</p>
          <UiStepper v-model="pax" :min="1" :max="Math.max(1, target.pax)" size="lg" label="Jumlah orang yang datang" />
        </div>
      </div>
      <template #footer>
        <UiButton variant="ghost" :disabled="!!busyId" @click="dialogOpen = false">Batal</UiButton>
        <UiButton :loading="!!busyId" @click="doCheckin"><UiIcon name="check" :size="16" /> Tandai datang</UiButton>
      </template>
    </UiDialog>
  </div>
</template>
