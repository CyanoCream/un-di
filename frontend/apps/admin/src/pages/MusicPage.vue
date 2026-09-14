<script setup lang="ts">
import { api, formatDate, type MusicTrack } from '@undangan/shared'
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import UiSpinner from '../components/UiSpinner.vue'
import UiState from '../components/UiState.vue'
import { confirmDialog } from '../lib/confirm'
import { toast } from '../lib/toast'
import { errMsg, fieldErrors, formatDuration, itemsOf } from '../lib/util'

const MAX_BYTES = 10 * 1024 * 1024

const tracks = ref<MusicTrack[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    tracks.value = itemsOf(await api.get<{ items: MusicTrack[] }>('/admin/music'))
  } catch (e) {
    error.value = errMsg(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ---- Upload ----
const form = reactive({ title: '', artist: '' })
const file = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const errors = ref<Record<string, string>>({})
const uploading = ref(false)
const dragOver = ref(false)

function pickFile(f: File | null | undefined) {
  errors.value = {}
  if (!f) return
  const isMp3 = f.type === 'audio/mpeg' || f.type === 'audio/mp3' || /\.mp3$/i.test(f.name)
  if (!isMp3) {
    errors.value.file = 'File harus berformat MP3.'
    return
  }
  if (f.size > MAX_BYTES) {
    errors.value.file = `Ukuran file ${(f.size / 1024 / 1024).toFixed(1)} MB melebihi batas 10 MB.`
    return
  }
  file.value = f
  if (!form.title) form.title = f.name.replace(/\.mp3$/i, '').replace(/[_]+/g, ' ').trim()
}

function onDrop(e: DragEvent) {
  dragOver.value = false
  pickFile(e.dataTransfer?.files?.[0])
}

function resetForm() {
  form.title = ''
  form.artist = ''
  file.value = null
  if (fileInput.value) fileInput.value.value = ''
}

async function upload() {
  errors.value = {}
  if (!file.value) errors.value.file = 'Pilih file MP3.'
  if (!form.title.trim()) errors.value.title = 'Judul wajib diisi.'
  if (Object.keys(errors.value).length) return
  uploading.value = true
  try {
    const track = await api.upload<MusicTrack>('/admin/music', file.value!, { title: form.title.trim(), artist: form.artist.trim() })
    tracks.value.unshift(track)
    toast.success(`“${track.title}” ditambahkan ke pustaka.`)
    resetForm()
  } catch (e) {
    errors.value = fieldErrors(e)
    if (!Object.keys(errors.value).length) toast.error(errMsg(e))
  } finally {
    uploading.value = false
  }
}

// ---- Player ----
const audio = new Audio()
audio.preload = 'none'
const playingId = ref<string | null>(null)
const bufferingId = ref<string | null>(null)
const progress = ref(0)

audio.addEventListener('timeupdate', () => {
  progress.value = audio.duration ? audio.currentTime / audio.duration : 0
})
audio.addEventListener('playing', () => (bufferingId.value = null))
audio.addEventListener('ended', () => {
  playingId.value = null
  progress.value = 0
})
audio.addEventListener('error', () => {
  if (playingId.value) toast.error('Gagal memutar audio.')
  playingId.value = null
  bufferingId.value = null
})

async function toggle(t: MusicTrack) {
  if (playingId.value === t.id) {
    audio.pause()
    playingId.value = null
    return
  }
  if (!audio.src.endsWith(t.url)) {
    audio.src = t.url
    progress.value = 0
  }
  playingId.value = t.id
  bufferingId.value = t.id
  try {
    await audio.play()
  } catch {
    if (playingId.value === t.id) {
      playingId.value = null
      bufferingId.value = null
    }
  }
}
onBeforeUnmount(() => {
  audio.pause()
  audio.src = ''
})

// ---- Hapus ----
const deleting = ref<string | null>(null)
async function remove(t: MusicTrack) {
  const ok = await confirmDialog({
    title: `Hapus “${t.title}”?`,
    message: 'Lagu dihapus dari pustaka musik.',
    details: ['Undangan yang sudah memakai lagu ini bisa kehilangan musik latar.', 'Tindakan tidak bisa dibatalkan.'],
    confirmText: 'Hapus',
    tone: 'danger',
  })
  if (!ok) return
  deleting.value = t.id
  try {
    await api.del(`/admin/music/${t.id}`)
    if (playingId.value === t.id) {
      audio.pause()
      playingId.value = null
    }
    tracks.value = tracks.value.filter((x) => x.id !== t.id)
    toast.success('Lagu dihapus.')
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    deleting.value = null
  }
}
</script>

<template>
  <div>
    <PageHeader title="Pustaka Musik" subtitle="Lagu latar yang bisa dipilih customer untuk undangan." />

    <div class="grid gap-5 lg:grid-cols-3">
      <!-- Upload -->
      <section class="card h-fit lg:col-span-1" aria-labelledby="upload-title">
        <div class="card-header"><h2 id="upload-title" class="card-title">Unggah lagu</h2></div>
        <form class="space-y-3.5 p-4" novalidate @submit.prevent="upload">
          <div>
            <span class="label">File MP3 (maks. 10 MB)</span>
            <label
              class="flex cursor-pointer flex-col items-center justify-center gap-1.5 rounded-lg border border-dashed px-3 py-6 text-center text-xs transition-colors"
              :class="[
                dragOver ? 'border-accent bg-accent/5' : errors.file ? 'border-danger/60' : 'border-line hover:border-muted',
                file ? 'text-ink' : 'text-muted',
              ]"
              @dragover.prevent="dragOver = true"
              @dragleave="dragOver = false"
              @drop.prevent="onDrop"
            >
              <AppIcon :name="file ? 'music' : 'upload'" :size="20" :class="file ? 'text-accent' : ''" />
              <span v-if="file" class="max-w-full truncate font-medium">{{ file.name }}</span>
              <span v-if="file" class="num text-muted">{{ (file.size / 1024 / 1024).toFixed(2) }} MB</span>
              <span v-else>Seret file ke sini atau <span class="text-accent">pilih file</span></span>
              <input ref="fileInput" type="file" accept="audio/mpeg,.mp3" class="sr-only" @change="pickFile(($event.target as HTMLInputElement).files?.[0])" />
            </label>
            <p v-if="errors.file" class="field-error">{{ errors.file }}</p>
          </div>
          <div>
            <label for="m-title" class="label">Judul</label>
            <input id="m-title" v-model="form.title" class="input" :class="errors.title && 'input-error'" maxlength="120" />
            <p v-if="errors.title" class="field-error">{{ errors.title }}</p>
          </div>
          <div>
            <label for="m-artist" class="label">Artis / sumber</label>
            <input id="m-artist" v-model="form.artist" class="input" maxlength="120" placeholder="mis. Pixabay Music" />
          </div>
          <div class="flex items-start gap-2 rounded-md border border-warning/25 bg-warning/5 px-3 py-2 text-xs text-warning/90">
            <AppIcon name="alert" :size="14" class="mt-px shrink-0" />
            <span>Unggah hanya lagu <b>royalty-free</b> atau yang lisensinya Anda miliki. Lagu komersial berhak cipta berisiko klaim pada layanan berbayar.</span>
          </div>
          <div class="flex justify-end gap-2">
            <button v-if="file || form.title || form.artist" type="button" class="btn btn-ghost" :disabled="uploading" @click="resetForm">Reset</button>
            <button type="submit" class="btn btn-primary" :disabled="uploading">
              <UiSpinner v-if="uploading" :size="14" /><AppIcon v-else name="upload" :size="14" /> {{ uploading ? 'Mengunggah…' : 'Unggah' }}
            </button>
          </div>
        </form>
      </section>

      <!-- Daftar -->
      <section class="card lg:col-span-2" aria-labelledby="list-title">
        <div class="card-header">
          <h2 id="list-title" class="card-title">Daftar lagu <span class="num ml-1 text-muted">{{ tracks.length }}</span></h2>
        </div>
        <UiState :loading="loading" :error="error" :empty="!tracks.length" empty-title="Pustaka masih kosong" empty-text="Unggah lagu royalty-free pertama." empty-icon="music" @retry="load">
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th class="w-10"><span class="sr-only">Putar</span></th>
                  <th>Judul</th>
                  <th class="text-right">Durasi</th>
                  <th>Ditambahkan</th>
                  <th class="w-10"><span class="sr-only">Aksi</span></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="t in tracks" :key="t.id" :class="playingId === t.id && 'bg-accent/5'">
                  <td>
                    <button
                      type="button"
                      class="grid size-7 place-items-center rounded-full transition-colors"
                      :class="playingId === t.id ? 'bg-accent text-accent-ink' : 'bg-surface-2 text-ink ring-1 ring-line hover:bg-line'"
                      :aria-label="playingId === t.id ? `Jeda ${t.title}` : `Putar ${t.title}`"
                      @click="toggle(t)"
                    >
                      <UiSpinner v-if="bufferingId === t.id" :size="12" />
                      <AppIcon v-else :name="playingId === t.id ? 'pause' : 'play'" :size="12" :stroke="0" class="fill-current" />
                    </button>
                  </td>
                  <td class="max-w-80">
                    <p class="truncate font-medium">{{ t.title }}</p>
                    <p class="truncate text-xs text-muted">{{ t.artist || '—' }}</p>
                    <div v-if="playingId === t.id" class="mt-1 h-0.5 w-full overflow-hidden rounded bg-line">
                      <div class="h-full bg-accent transition-[width]" :style="{ width: `${progress * 100}%` }" />
                    </div>
                  </td>
                  <td class="num text-right text-muted">{{ formatDuration(t.duration_seconds) }}</td>
                  <td class="num text-muted">{{ formatDate(t.created_at) }}</td>
                  <td>
                    <button
                      type="button"
                      class="btn btn-ghost btn-sm btn-icon hover:text-danger"
                      :aria-label="`Hapus ${t.title}`"
                      :disabled="deleting === t.id"
                      @click="remove(t)"
                    >
                      <UiSpinner v-if="deleting === t.id" :size="12" /><AppIcon v-else name="trash" :size="14" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </UiState>
      </section>
    </div>
  </div>
</template>
