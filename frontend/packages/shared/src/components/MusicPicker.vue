<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '../api'
import { toast } from '../composables/useToast'
import type { InvitationContent, MusicTrack } from '../types'
import { errorMessage } from '../utils/misc'
import UiIcon from './ui/UiIcon.vue'
import UiInput from './ui/UiInput.vue'
import UiSpinner from './ui/UiSpinner.vue'
import UiSwitch from './ui/UiSwitch.vue'

type Music = InvitationContent['music']

const props = defineProps<{ modelValue: Music }>()
const emit = defineEmits<{ 'update:modelValue': [value: Music] }>()

const MAX_MB = 10

function update(patch: Partial<Music>) {
  emit('update:modelValue', { ...props.modelValue, ...patch })
}

// ---- Pustaka ----
const tracks = ref<MusicTrack[]>([])
const loading = ref(false)
const loadError = ref('')
const q = ref('')

async function load() {
  if (cache) {
    tracks.value = cache
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    const res = await api.get<{ items: MusicTrack[] }>('/music')
    tracks.value = res.items ?? []
    cache = tracks.value
  } catch (e) {
    loadError.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => {
  const s = q.value.trim().toLowerCase()
  if (!s) return tracks.value
  return tracks.value.filter((t) => `${t.title} ${t.artist}`.toLowerCase().includes(s))
})

const isCustom = computed(() => !!props.modelValue.url && !tracks.value.some((t) => t.url === props.modelValue.url))
const tab = ref<'library' | 'upload'>('library')

// ---- Pemutar pratinjau (satu <audio>) ----
const audio = ref<HTMLAudioElement | null>(null)
const playingUrl = ref('')
const audioLoading = ref(false)

function toggle(url: string, startAt = 0) {
  const el = audio.value
  if (!el) return
  if (playingUrl.value === url && !el.paused) {
    el.pause()
    playingUrl.value = ''
    return
  }
  if (el.getAttribute('src') !== url) {
    el.src = url
    audioLoading.value = true
  }
  el.currentTime = startAt
  playingUrl.value = url
  el.play().catch(() => {
    audioLoading.value = false
    playingUrl.value = ''
    toast.error('Lagu tidak dapat diputar')
  })
}

function onCanPlay() {
  audioLoading.value = false
}
function onEnded() {
  playingUrl.value = ''
}
function onLoadedMeta() {
  const el = audio.value
  if (el && playingUrl.value === props.modelValue.url && props.modelValue.start_at) {
    el.currentTime = Math.min(props.modelValue.start_at, Math.max(0, el.duration - 1))
  }
}

function choose(t: MusicTrack) {
  update({ url: t.url, title: t.artist ? `${t.title} — ${t.artist}` : t.title, enabled: true })
}

function fmtDur(s: number) {
  if (!s) return ''
  const m = Math.floor(s / 60)
  return `${m}:${String(Math.floor(s % 60)).padStart(2, '0')}`
}

// ---- Upload sendiri ----
const fileInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)

async function onFile(e: Event) {
  const el = e.target as HTMLInputElement
  const file = el.files?.[0]
  el.value = ''
  if (!file) return
  const isMp3 = file.type === 'audio/mpeg' || file.type === 'audio/mp3' || /\.mp3$/i.test(file.name)
  if (!isMp3) {
    toast.error('Format lagu harus MP3')
    return
  }
  if (file.size > MAX_MB * 1024 * 1024) {
    toast.error(`Ukuran lagu maksimal ${MAX_MB} MB`)
    return
  }
  uploading.value = true
  try {
    const res = await api.upload<{ url: string }>('/uploads', file, { kind: 'audio' })
    update({ url: res.url, title: file.name.replace(/\.[^.]+$/, ''), enabled: true })
    toast.success('Lagu berhasil diunggah')
  } catch (err) {
    toast.error(errorMessage(err))
  } finally {
    uploading.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => audio.value?.pause())
</script>

<script lang="ts">
let cache: MusicTrack[] | null = null
</script>

<template>
  <div class="space-y-4">
    <audio ref="audio" preload="none" class="hidden" @canplay="onCanPlay" @ended="onEnded" @loadedmetadata="onLoadedMeta" />

    <UiSwitch
      :model-value="modelValue.enabled"
      label="Putar musik latar"
      description="Musik mulai setelah tamu menekan “Buka Undangan”."
      @update:model-value="update({ enabled: $event })"
    />

    <div v-if="modelValue.enabled" class="space-y-4">
      <!-- Lagu terpilih -->
      <div class="flex items-center gap-3 rounded-xl border border-line bg-surface-2 p-3">
        <button
          type="button"
          class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent text-accent-ink disabled:opacity-40"
          :disabled="!modelValue.url"
          :aria-label="playingUrl === modelValue.url ? 'Jeda' : 'Putar'"
          @click="toggle(modelValue.url, modelValue.start_at)"
        >
          <UiSpinner v-if="audioLoading && playingUrl === modelValue.url" :size="16" />
          <UiIcon v-else :name="playingUrl === modelValue.url && modelValue.url ? 'pause' : 'play'" :size="16" />
        </button>
        <div class="min-w-0 flex-1">
          <p class="text-xs text-muted">Lagu terpilih</p>
          <p class="truncate text-sm font-medium text-ink">{{ modelValue.url ? modelValue.title || 'Tanpa judul' : 'Belum ada lagu' }}</p>
        </div>
        <button
          v-if="modelValue.url"
          type="button"
          class="rounded-lg p-2 text-muted hover:bg-surface hover:text-danger"
          aria-label="Hapus lagu"
          @click="update({ url: '', title: '' })"
        >
          <UiIcon name="x" :size="16" />
        </button>
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-ink">Judul lagu</span>
          <UiInput :model-value="modelValue.title" placeholder="Judul yang ditampilkan" @update:model-value="update({ title: String($event) })" />
        </label>
        <label class="block">
          <span class="mb-1.5 block text-sm font-medium text-ink">Mulai dari detik ke-</span>
          <UiInput
            type="number"
            min="0"
            :model-value="modelValue.start_at"
            @update:model-value="update({ start_at: Math.max(0, Math.floor(Number($event) || 0)) })"
          />
        </label>
      </div>

      <div class="flex gap-1 rounded-xl bg-surface-2 p-1 text-sm">
        <button
          type="button"
          class="flex-1 rounded-lg px-3 py-1.5 font-medium transition"
          :class="tab === 'library' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'"
          @click="tab = 'library'"
        >
          Pustaka musik
        </button>
        <button
          type="button"
          class="flex-1 rounded-lg px-3 py-1.5 font-medium transition"
          :class="tab === 'upload' ? 'bg-surface text-ink shadow-sm' : 'text-muted hover:text-ink'"
          @click="tab = 'upload'"
        >
          Unggah sendiri
        </button>
      </div>

      <div v-if="tab === 'library'">
        <UiInput v-if="tracks.length > 6" v-model="q" placeholder="Cari lagu…" class="mb-2" />
        <div v-if="loading" class="flex items-center justify-center gap-2 py-8 text-sm text-muted"><UiSpinner /> Memuat pustaka…</div>
        <div v-else-if="loadError" class="py-6 text-center text-sm text-danger">
          {{ loadError }}
          <button type="button" class="ml-1 underline" @click="load">Coba lagi</button>
        </div>
        <p v-else-if="!filtered.length" class="py-6 text-center text-sm text-muted">Belum ada lagu di pustaka.</p>
        <ul v-else class="max-h-80 divide-y divide-line overflow-y-auto rounded-xl border border-line bg-surface">
          <li v-for="t in filtered" :key="t.id" class="flex items-center gap-3 px-3 py-2.5">
            <button
              type="button"
              class="flex size-8 shrink-0 items-center justify-center rounded-full border border-line text-ink hover:bg-surface-2"
              :aria-label="playingUrl === t.url ? 'Jeda' : 'Putar pratinjau'"
              @click="toggle(t.url)"
            >
              <UiSpinner v-if="audioLoading && playingUrl === t.url" :size="14" />
              <UiIcon v-else :name="playingUrl === t.url ? 'pause' : 'play'" :size="14" />
            </button>
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium text-ink">{{ t.title }}</p>
              <p class="truncate text-xs text-muted">
                {{ t.artist }}<template v-if="t.artist && t.duration_seconds"> · </template>{{ fmtDur(t.duration_seconds) }}
              </p>
            </div>
            <span
              v-if="modelValue.url === t.url"
              class="inline-flex items-center gap-1 rounded-full bg-accent/12 px-2.5 py-1 text-xs font-medium text-accent"
            >
              <UiIcon name="check" :size="13" /> Dipilih
            </span>
            <button
              v-else
              type="button"
              class="rounded-lg border border-line px-3 py-1 text-xs font-medium text-ink hover:bg-surface-2"
              @click="choose(t)"
            >
              Pilih
            </button>
          </li>
        </ul>
      </div>

      <div v-else class="space-y-3">
        <input ref="fileInput" type="file" accept=".mp3,audio/mpeg" class="hidden" @change="onFile" />
        <button
          type="button"
          class="flex w-full flex-col items-center justify-center gap-1.5 rounded-xl border border-dashed border-line bg-surface px-4 py-6 text-sm text-muted transition hover:border-accent hover:text-accent disabled:opacity-60"
          :disabled="uploading"
          @click="fileInput?.click()"
        >
          <UiSpinner v-if="uploading" :size="22" />
          <UiIcon v-else name="music" :size="22" />
          <span class="font-medium">{{ uploading ? 'Mengunggah lagu…' : 'Pilih file MP3' }}</span>
          <span class="text-xs">Maksimal {{ MAX_MB }} MB</span>
        </button>
        <p v-if="isCustom" class="text-xs text-muted">Lagu saat ini: file unggahan sendiri.</p>
        <div class="flex gap-2 rounded-xl border border-warning/30 bg-warning/10 p-3 text-xs leading-relaxed text-ink">
          <UiIcon name="alert" :size="16" class="mt-0.5 text-warning" />
          <p>
            Dengan mengunggah lagu, Anda bertanggung jawab penuh atas hak cipta lagu tersebut. Gunakan lagu yang Anda miliki
            haknya atau yang berlisensi bebas royalti.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
