<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../api'
import { confirm } from '../composables/useConfirm'
import { useMediaQuery } from '../composables/useMediaQuery'
import { THEME_CATEGORIES, type ThemeCategory, type ThemeInfo } from '../types'
import PreviewFrame from './PreviewFrame.vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiDialog from './ui/UiDialog.vue'
import UiEmpty from './ui/UiEmpty.vue'
import UiIcon from './ui/UiIcon.vue'

const props = withDefaults(
  defineProps<{
    themes: ThemeInfo[]
    current: string | null
    locked: boolean
    canOverrideLock: boolean
    invitationId?: string
    busy?: boolean
    /**
     * Slug tema yang disorot & di-scroll ke tampilan saat pertama tampil
     * (mis. tema yang dipilih pengunjung di landing page). Tidak memilih otomatis.
     */
    highlight?: string | null
  }>(),
  { invitationId: undefined, busy: false, highlight: null },
)

const emit = defineEmits<{ select: [slug: string] }>()

const blocked = computed(() => props.locked && !props.canOverrideLock)
const pendingSlug = ref<string | null>(null)
watch(
  () => props.busy,
  (b) => {
    if (!b) pendingSlug.value = null
  },
)

const sorted = computed(() => [...props.themes].sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name)))

// ---------- Filter kategori & pencarian ----------
const CATEGORY_LABEL = Object.fromEntries(THEME_CATEGORIES.map((c) => [c.value, c.label])) as Record<ThemeCategory, string>

const category = ref<ThemeCategory | 'all'>('all')
const query = ref('')

const categoryChips = computed(() => {
  const counts = new Map<string, number>()
  for (const t of props.themes) if (t.category) counts.set(t.category, (counts.get(t.category) ?? 0) + 1)
  return THEME_CATEGORIES.filter((c) => counts.has(c.value)).map((c) => ({ ...c, count: counts.get(c.value) ?? 0 }))
})

const chips = computed<{ value: ThemeCategory | 'all'; label: string; count: number }[]>(() => [
  { value: 'all', label: 'Semua', count: props.themes.length },
  ...categoryChips.value,
])

// Kategori terpilih hilang dari daftar (mis. tema dinonaktifkan) → kembali ke Semua.
watch(categoryChips, (chips) => {
  if (category.value !== 'all' && !chips.some((c) => c.value === category.value)) category.value = 'all'
})

const normalizedQuery = computed(() => query.value.trim().toLowerCase())
const filtered = computed(() => {
  const q = normalizedQuery.value
  return sorted.value.filter((t) => (category.value === 'all' || t.category === category.value) && (!q || t.name.toLowerCase().includes(q)))
})
const isFiltering = computed(() => category.value !== 'all' || !!normalizedQuery.value)
const countLabel = computed(() =>
  isFiltering.value ? `${filtered.value.length} dari ${sorted.value.length} tema` : `${sorted.value.length} tema`,
)

function resetFilter() {
  category.value = 'all'
  query.value = ''
}

// ---------- Thumbnail ----------
const brokenThumbs = reactive(new Set<string>())
const hasThumb = (t: ThemeInfo) => !!t.thumbnail_url && !brokenThumbs.has(t.slug)
const swatchColors = (t: ThemeInfo) => (t.colors.length ? t.colors : ['#e5e5e5', '#d4d4d4'])

// ---------- Sorotan (tema dari landing page) ----------
const highlightTheme = computed(() => (props.highlight ? (props.themes.find((t) => t.slug === props.highlight) ?? null) : null))
const gridEl = ref<HTMLElement | null>(null)
let scrolledTo: string | null = null

async function scrollToHighlight(force = false) {
  const t = highlightTheme.value
  if (!t || (!force && scrolledTo === t.slug)) return
  if (!filtered.value.some((x) => x.slug === t.slug)) resetFilter()
  await nextTick()
  const el = gridEl.value?.querySelector<HTMLElement>(`[data-slug="${CSS.escape(t.slug)}"]`)
  if (!el) return
  scrolledTo = t.slug
  const reduce = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  el.scrollIntoView({ block: 'center', behavior: reduce ? 'auto' : 'smooth' })
}

onMounted(() => scrollToHighlight())
watch(highlightTheme, () => scrollToHighlight())

// ---------- Pratinjau & pilih ----------
const previewTheme = ref<ThemeInfo | null>(null)
const previewOpen = computed({
  get: () => !!previewTheme.value,
  set: (v: boolean) => {
    if (!v) previewTheme.value = null
  },
})
const device = ref<'mobile' | 'desktop'>('mobile')
const isSmall = useMediaQuery('(max-width: 639px)')

function previewSrc(t: ThemeInfo) {
  if (props.invitationId) return api.url(`/invitations/${props.invitationId}/preview`, { theme: t.slug })
  return t.preview_url
}

async function choose(t: ThemeInfo) {
  if (blocked.value || props.busy || t.slug === props.current) return
  if (!props.canOverrideLock) {
    const ok = await confirm({
      title: `Pilih tema ${t.name}?`,
      message: 'Tema hanya bisa dipilih satu kali. Setelah dipilih tidak bisa diganti sendiri. Lanjutkan?',
      confirmText: 'Ya, pilih tema ini',
    })
    if (!ok) return
  }
  pendingSlug.value = t.slug
  previewTheme.value = null
  emit('select', t.slug)
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="blocked" class="flex items-start gap-3 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-ink">
      <UiIcon name="lock" :size="18" class="mt-0.5 text-warning" />
      <p>Tema sudah dipilih dan terkunci. Hubungi admin untuk mengganti.</p>
    </div>
    <div
      v-else-if="!canOverrideLock && !locked"
      class="flex items-start gap-3 rounded-xl border border-line bg-surface-2 px-4 py-3 text-sm text-muted"
    >
      <UiIcon name="info" :size="18" class="mt-0.5 text-accent" />
      <p>Lihat pratinjau dulu dengan data undangan Anda. Tema hanya dapat dipilih <strong class="text-ink">satu kali</strong>.</p>
    </div>
    <div v-else-if="canOverrideLock && locked" class="flex items-start gap-3 rounded-xl border border-line bg-surface-2 px-4 py-3 text-sm text-muted">
      <UiIcon name="lock" :size="18" class="mt-0.5" />
      <p>Tema terkunci untuk customer. Sebagai admin Anda tetap dapat menggantinya.</p>
    </div>

    <div
      v-if="highlightTheme && highlightTheme.slug !== current && !blocked"
      class="flex flex-wrap items-center gap-3 rounded-xl border border-accent/30 bg-accent/8 px-4 py-3 text-sm text-ink"
    >
      <UiIcon name="sparkles" :size="18" class="shrink-0 text-accent" />
      <p class="min-w-0 flex-1">
        Anda sebelumnya tertarik dengan tema <strong>{{ highlightTheme.name }}</strong>. Lihat pratinjaunya dengan data undangan Anda sebelum memilih.
      </p>
      <div class="flex gap-2">
        <UiButton variant="secondary" size="sm" @click="scrollToHighlight(true)">Tunjukkan</UiButton>
        <UiButton size="sm" @click="previewTheme = highlightTheme"><UiIcon name="eye" :size="14" /> Pratinjau</UiButton>
      </div>
    </div>

    <UiEmpty v-if="!sorted.length" title="Belum ada tema" description="Tema yang aktif akan tampil di sini." icon="palette" />

    <template v-else>
      <!-- Toolbar: cari + kategori -->
      <div class="space-y-3">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
          <label class="relative block min-w-0 flex-1 sm:max-w-xs">
            <span class="sr-only">Cari tema</span>
            <UiIcon name="search" :size="16" class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
            <input
              v-model="query"
              type="search"
              placeholder="Cari nama tema…"
              autocomplete="off"
              class="h-10 w-full rounded-xl border border-line bg-surface-2 pr-9 pl-9 text-sm text-ink outline-none transition placeholder:text-muted/70 focus:border-accent focus:bg-surface focus:ring-3 focus:ring-accent/15 [&::-webkit-search-cancel-button]:hidden"
            />
            <button
              v-if="query"
              type="button"
              class="absolute top-1/2 right-2 -translate-y-1/2 rounded-md p-1 text-muted hover:text-ink"
              aria-label="Hapus pencarian"
              @click="query = ''"
            >
              <UiIcon name="x" :size="14" />
            </button>
          </label>
          <p class="text-sm text-muted sm:ml-auto" aria-live="polite">{{ countLabel }}</p>
        </div>

        <div
          v-if="categoryChips.length > 1"
          class="-mx-1 flex gap-2 overflow-x-auto px-1 pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
          role="group"
          aria-label="Filter kategori tema"
        >
          <button
            v-for="chip in chips"
            :key="chip.value"
            type="button"
            class="inline-flex h-8 shrink-0 items-center gap-1.5 rounded-full border px-3 text-sm whitespace-nowrap transition"
            :class="category === chip.value ? 'border-accent bg-accent text-accent-ink' : 'border-line bg-surface text-ink hover:bg-surface-2'"
            :aria-pressed="category === chip.value"
            @click="category = chip.value"
          >
            {{ chip.label }}
            <span class="text-xs" :class="category === chip.value ? 'opacity-80' : 'text-muted'">{{ chip.count }}</span>
          </button>
        </div>
      </div>

      <UiEmpty v-if="!filtered.length" title="Tema tidak ditemukan" description="Coba kata kunci atau kategori lain." icon="search" compact>
        <UiButton variant="secondary" size="sm" @click="resetFilter">Tampilkan semua tema</UiButton>
      </UiEmpty>

      <div v-else ref="gridEl" class="grid grid-cols-2 gap-3 sm:gap-4 md:grid-cols-3 xl:grid-cols-4">
        <article
          v-for="t in filtered"
          :key="t.slug"
          :data-slug="t.slug"
          class="group flex scroll-mt-24 flex-col overflow-hidden rounded-2xl border bg-surface transition"
          :class="
            t.slug === current
              ? 'border-accent ring-2 ring-accent/25'
              : t.slug === highlight
                ? 'border-accent/60 ring-2 ring-accent/40 ring-offset-2 ring-offset-surface'
                : 'border-line hover:shadow-md'
          "
        >
          <button
            type="button"
            class="relative block aspect-[3/4] w-full overflow-hidden bg-surface-2 text-left"
            :aria-label="`Pratinjau ${t.name}`"
            @click="previewTheme = t"
          >
            <img
              v-if="hasThumb(t)"
              :src="t.thumbnail_url"
              :alt="`Tampilan tema ${t.name}`"
              loading="lazy"
              decoding="async"
              width="390"
              height="520"
              class="size-full object-cover object-top transition duration-500 group-hover:scale-[1.03]"
              @error="brokenThumbs.add(t.slug)"
            />
            <div v-else class="relative flex size-full" aria-hidden="true">
              <span v-for="(col, i) in swatchColors(t)" :key="i" class="h-full flex-1" :style="{ background: col }" />
              <span class="absolute inset-x-4 top-1/2 -translate-y-1/2 rounded-xl bg-surface/85 px-3 py-4 text-center shadow-sm backdrop-blur-sm">
                <span class="block text-[10px] tracking-[0.2em] text-muted uppercase">The Wedding of</span>
                <span class="mt-1 block truncate text-base font-semibold text-ink sm:text-lg" :style="{ fontFamily: t.fonts[0] ? `'${t.fonts[0]}', serif` : undefined }">
                  {{ t.name }}
                </span>
              </span>
            </div>

            <span class="absolute inset-0 flex items-center justify-center bg-ink/0 opacity-0 transition group-hover:bg-ink/15 group-hover:opacity-100">
              <span class="inline-flex items-center gap-1.5 rounded-full bg-surface/95 px-3 py-1.5 text-xs font-medium text-ink shadow">
                <UiIcon name="eye" :size="14" /> Pratinjau
              </span>
            </span>
            <span class="absolute top-2 right-2 left-2 flex flex-wrap gap-1.5">
              <UiBadge v-if="t.slug === current" tone="success" class="bg-surface!"><UiIcon name="check" :size="12" /> Dipakai</UiBadge>
              <UiBadge v-else-if="t.slug === highlight" tone="accent" class="bg-surface!"><UiIcon name="sparkles" :size="12" /> Pilihan Anda</UiBadge>
              <UiBadge v-if="t.is_premium" tone="warning" class="bg-surface!"><UiIcon name="crown" :size="12" /> Premium</UiBadge>
            </span>
          </button>

          <div class="flex flex-1 flex-col p-3 sm:p-4">
            <p v-if="t.category" class="truncate text-[11px] font-medium tracking-wide text-accent uppercase">{{ CATEGORY_LABEL[t.category] }}</p>
            <h3 class="truncate font-semibold text-ink">{{ t.name }}</h3>
            <p class="mt-1 line-clamp-2 text-xs text-muted sm:text-sm">{{ t.description }}</p>
            <div class="mt-auto flex flex-col gap-2 pt-3 sm:flex-row sm:pt-4">
              <UiButton variant="secondary" size="sm" class="sm:flex-1" @click="previewTheme = t">
                <UiIcon name="eye" :size="14" /> Pratinjau
              </UiButton>
              <UiButton
                v-if="t.slug !== current"
                size="sm"
                class="sm:flex-1"
                :disabled="blocked || (busy && pendingSlug !== t.slug)"
                :loading="busy && pendingSlug === t.slug"
                @click="choose(t)"
              >
                <UiIcon v-if="blocked" name="lock" :size="14" />
                {{ canOverrideLock && current ? 'Ganti ke sini' : 'Pilih' }}
              </UiButton>
              <span v-else class="inline-flex h-8 items-center justify-center gap-1 text-sm font-medium text-success sm:flex-1">
                <UiIcon name="check-circle" :size="15" /> Tema aktif
              </span>
            </div>
          </div>
        </article>
      </div>
    </template>

    <UiDialog v-model:open="previewOpen" size="full" flush :title="previewTheme ? `Pratinjau: ${previewTheme.name}` : ''">
      <template #actions>
        <div class="hidden items-center gap-1 rounded-lg bg-surface-2 p-0.5 sm:flex">
          <button
            type="button"
            class="rounded-md p-1.5 transition"
            :class="device === 'mobile' ? 'bg-surface text-ink shadow-sm' : 'text-muted'"
            aria-label="Tampilan ponsel"
            @click="device = 'mobile'"
          >
            <UiIcon name="smartphone" :size="16" />
          </button>
          <button
            type="button"
            class="rounded-md p-1.5 transition"
            :class="device === 'desktop' ? 'bg-surface text-ink shadow-sm' : 'text-muted'"
            aria-label="Tampilan desktop"
            @click="device = 'desktop'"
          >
            <UiIcon name="monitor" :size="16" />
          </button>
        </div>
      </template>
      <div v-if="previewTheme" class="h-full bg-surface-2 p-0 sm:p-4">
        <PreviewFrame :src="previewSrc(previewTheme)" :device="isSmall ? 'desktop' : device" />
      </div>
      <template v-if="previewTheme" #footer>
        <p v-if="!invitationId" class="mr-auto text-xs text-muted">Pratinjau memakai data contoh.</p>
        <p v-else class="mr-auto text-xs text-muted">Pratinjau memakai data undangan Anda.</p>
        <UiButton variant="ghost" @click="previewTheme = null">Tutup</UiButton>
        <UiButton
          v-if="previewTheme.slug !== current && !blocked"
          :disabled="busy"
          @click="choose(previewTheme)"
        >
          {{ canOverrideLock && current ? 'Ganti ke tema ini' : 'Pilih tema ini' }}
        </UiButton>
      </template>
    </UiDialog>
  </div>
</template>
