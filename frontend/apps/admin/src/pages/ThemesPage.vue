<script setup lang="ts">
import { api, THEME_CATEGORIES, type ThemeCategory, type ThemeInfo } from '@undangan/shared'
import { computed, onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import UiBadge from '../components/UiBadge.vue'
import UiDialog from '../components/UiDialog.vue'
import UiState from '../components/UiState.vue'
import UiToggle from '../components/UiToggle.vue'
import { toast } from '../lib/toast'
import { errMsg, itemsOf } from '../lib/util'

const themes = ref<ThemeInfo[]>([])
const loading = ref(true)
const error = ref('')
const saving = ref<Record<string, boolean>>({})
const filter = ref<'all' | 'active' | 'inactive'>('all')
/** '' = semua kategori, 'none' = tema tanpa kategori. */
const categoryFilter = ref<ThemeCategory | '' | 'none'>('')

const CATEGORY_LABEL = Object.fromEntries(THEME_CATEGORIES.map((c) => [c.value, c.label])) as Record<ThemeCategory, string>
const categoryOptions = computed(() => {
  const counts = new Map<string, number>()
  for (const t of themes.value) counts.set(t.category || 'none', (counts.get(t.category || 'none') ?? 0) + 1)
  const opts: { value: ThemeCategory | 'none'; label: string }[] = THEME_CATEGORIES.filter((c) => counts.has(c.value)).map((c) => ({
    value: c.value,
    label: `${c.label} (${counts.get(c.value)})`,
  }))
  if (counts.has('none')) opts.push({ value: 'none', label: `Tanpa kategori (${counts.get('none')})` })
  return opts
})

// Thumbnail gagal dimuat → tampilkan swatch warna.
const brokenThumbs = reactive(new Set<string>())
const hasThumb = (t: ThemeInfo) => !!t.thumbnail_url && !brokenThumbs.has(t.slug)

async function load() {
  loading.value = true
  error.value = ''
  try {
    themes.value = itemsOf(await api.get<{ items: ThemeInfo[] }>('/admin/themes'))
  } catch (e) {
    error.value = errMsg(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)

const visible = computed(() =>
  [...themes.value]
    .filter((t) => (filter.value === 'all' ? true : filter.value === 'active' ? t.is_active : !t.is_active))
    .filter((t) => (!categoryFilter.value ? true : categoryFilter.value === 'none' ? !t.category : t.category === categoryFilter.value))
    .sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name)),
)
const activeCount = computed(() => themes.value.filter((t) => t.is_active).length)

async function patch(t: ThemeInfo, change: Partial<Pick<ThemeInfo, 'is_active' | 'is_premium' | 'sort_order'>>) {
  const prev = { is_active: t.is_active, is_premium: t.is_premium, sort_order: t.sort_order }
  Object.assign(t, change)
  saving.value[t.slug] = true
  try {
    const updated = await api.patch<ThemeInfo>(`/admin/themes/${t.slug}`, {
      is_active: t.is_active,
      is_premium: t.is_premium,
      sort_order: t.sort_order,
    })
    Object.assign(t, updated)
    toast.success(`Tema ${t.name} diperbarui.`)
  } catch (e) {
    Object.assign(t, prev)
    toast.error(errMsg(e))
  } finally {
    saving.value[t.slug] = false
  }
}

function onSortChange(t: ThemeInfo, e: Event) {
  const v = Number((e.target as HTMLInputElement).value)
  if (!Number.isInteger(v) || v === t.sort_order) {
    ;(e.target as HTMLInputElement).value = String(t.sort_order)
    return
  }
  patch(t, { sort_order: v })
}

// Pratinjau
const preview = ref<ThemeInfo | null>(null)
const device = ref<'mobile' | 'desktop'>('mobile')
const previewUrl = (t: ThemeInfo) => t.preview_url || `/_preview/${t.slug}`
</script>

<template>
  <div>
    <PageHeader title="Tema" :subtitle="`${activeCount} dari ${themes.length} tema aktif. Tema nonaktif tidak bisa dipilih customer.`">
      <template #actions>
        <select v-model="categoryFilter" class="input h-8 w-auto" aria-label="Filter kategori">
          <option value="">Semua kategori</option>
          <option v-for="o in categoryOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
        </select>
        <select v-model="filter" class="input h-8 w-auto" aria-label="Filter tema">
          <option value="all">Semua tema</option>
          <option value="active">Aktif</option>
          <option value="inactive">Nonaktif</option>
        </select>
      </template>
    </PageHeader>

    <UiState :loading="loading" :error="error" :empty="!visible.length" empty-title="Tidak ada tema" empty-icon="palette" @retry="load">
      <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <article v-for="t in visible" :key="t.slug" class="card flex flex-col overflow-hidden" :class="!t.is_active && 'opacity-70'">
          <div class="flex gap-3 p-4 pb-0">
            <!-- Thumbnail / swatch -->
            <button
              type="button"
              class="group relative aspect-[3/4] w-24 shrink-0 overflow-hidden rounded-md bg-surface-2 ring-1 ring-line sm:w-28"
              :aria-label="`Pratinjau tema ${t.name}`"
              @click="preview = t"
            >
              <img
                v-if="hasThumb(t)"
                :src="t.thumbnail_url"
                :alt="`Thumbnail ${t.name}`"
                loading="lazy"
                decoding="async"
                width="390"
                height="520"
                class="size-full object-cover object-top"
                @error="brokenThumbs.add(t.slug)"
              />
              <div v-else class="flex size-full" :title="t.thumbnail_url ? 'Thumbnail gagal dimuat' : 'Belum ada thumbnail'">
                <span v-for="(c, i) in t.colors" :key="i" class="h-full flex-1" :style="{ background: c }" />
                <span v-if="!t.colors?.length" class="h-full flex-1 bg-surface-2" />
              </div>
              <span class="absolute inset-0 grid place-items-center bg-black/0 text-white opacity-0 transition group-hover:bg-black/45 group-hover:opacity-100">
                <AppIcon name="eye" />
              </span>
            </button>

            <div class="flex min-w-0 flex-1 flex-col gap-2">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <h2 class="truncate font-semibold">{{ t.name }}</h2>
                  <UiBadge v-if="t.is_premium" tone="warning" :dot="false" class="shrink-0"><AppIcon name="star" :size="10" /> Premium</UiBadge>
                </div>
                <code class="block truncate font-mono text-[11px] text-muted">{{ t.slug }}</code>
              </div>
              <div class="flex flex-wrap items-center gap-1.5">
                <UiBadge v-if="t.category" tone="accent" :dot="false">{{ CATEGORY_LABEL[t.category] ?? t.category }}</UiBadge>
                <UiBadge v-else tone="neutral" :dot="false">Tanpa kategori</UiBadge>
                <span v-if="!t.thumbnail_url" class="inline-flex items-center gap-1 text-[11px] text-warning"><AppIcon name="image" :size="11" /> Tanpa thumbnail</span>
              </div>
              <p class="line-clamp-2 text-xs text-muted">{{ t.description }}</p>
              <div class="flex flex-wrap gap-1">
                <span v-for="tag in t.tags" :key="tag" class="rounded bg-surface-2 px-1.5 py-0.5 text-[11px] text-muted">{{ tag }}</span>
              </div>
              <div class="flex flex-wrap items-center gap-2 text-[11px] text-muted">
                <span class="inline-flex items-center gap-1">
                  <span v-for="(c, i) in t.colors" :key="i" class="size-3 rounded-full ring-1 ring-line" :style="{ background: c }" :title="c" />
                </span>
                <span v-if="t.fonts?.length" class="truncate">· {{ t.fonts.join(', ') }}</span>
              </div>
            </div>
          </div>

          <div class="flex flex-1 flex-col gap-3 p-4">
            <div class="mt-auto flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-line pt-3">
              <UiToggle :model-value="t.is_active" label="Aktif" tone="success" :disabled="saving[t.slug]" @update:model-value="patch(t, { is_active: $event })" />
              <UiToggle :model-value="t.is_premium" label="Premium" tone="warning" :disabled="saving[t.slug]" @update:model-value="patch(t, { is_premium: $event })" />
              <label class="ml-auto flex items-center gap-1.5 text-xs text-muted">
                Urutan
                <input
                  type="number"
                  class="input num h-7 w-16 px-2 text-xs"
                  :value="t.sort_order"
                  :disabled="saving[t.slug]"
                  @change="onSortChange(t, $event)"
                  @keydown.enter="($event.target as HTMLInputElement).blur()"
                />
              </label>
            </div>
            <div class="flex gap-2">
              <button type="button" class="btn btn-secondary btn-sm flex-1" @click="preview = t"><AppIcon name="eye" :size="13" /> Pratinjau</button>
              <a :href="previewUrl(t)" target="_blank" rel="noopener" class="btn btn-ghost btn-sm"><AppIcon name="external" :size="13" /> Tab baru</a>
            </div>
          </div>
        </article>
      </div>
    </UiState>

    <UiDialog :open="!!preview" size="full" @close="preview = null">
      <template #header>
        <div class="flex min-w-0 flex-1 flex-wrap items-center gap-3">
          <h2 class="truncate text-[15px] font-semibold">Pratinjau · {{ preview?.name }}</h2>
          <div class="flex rounded-md bg-canvas p-0.5 ring-1 ring-line" role="group" aria-label="Ukuran layar">
            <button
              v-for="d in (['mobile', 'desktop'] as const)"
              :key="d"
              type="button"
              class="inline-flex h-7 items-center gap-1 rounded px-2 text-xs"
              :class="device === d ? 'bg-surface-2 text-ink' : 'text-muted hover:text-ink'"
              :aria-pressed="device === d"
              @click="device = d"
            >
              <AppIcon :name="d === 'mobile' ? 'phone' : 'monitor'" :size="13" /> {{ d === 'mobile' ? 'Ponsel' : 'Desktop' }}
            </button>
          </div>
          <a v-if="preview" :href="previewUrl(preview)" target="_blank" rel="noopener" class="btn btn-ghost btn-sm ml-auto"><AppIcon name="external" :size="13" /> Tab baru</a>
        </div>
      </template>
      <div class="flex justify-center rounded-lg bg-canvas p-3">
        <iframe
          v-if="preview"
          :key="preview.slug"
          :src="previewUrl(preview)"
          :title="`Pratinjau tema ${preview.name}`"
          class="h-[70vh] rounded-md border border-line bg-white transition-[width]"
          :class="device === 'mobile' ? 'w-[390px] max-w-full' : 'w-full'"
          loading="lazy"
        />
      </div>
    </UiDialog>
  </div>
</template>
