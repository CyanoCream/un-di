<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../api'
import { toast } from '../composables/useToast'
import { compressImage } from '../utils/image'
import { errorMessage } from '../utils/misc'
import UiIcon from './ui/UiIcon.vue'
import UiSpinner from './ui/UiSpinner.vue'

const props = withDefaults(
  defineProps<{
    modelValue: string | string[]
    multiple?: boolean
    max?: number
    /** CSS aspect-ratio untuk tile, mis. "1 / 1", "3 / 4". */
    aspect?: string
    /** Batas ukuran file asli (MB) sebelum kompres. */
    maxSizeMb?: number
    disabled?: boolean
  }>(),
  { multiple: false, max: 20, aspect: '1 / 1', maxSizeMb: 15, disabled: false },
)

const emit = defineEmits<{ 'update:modelValue': [value: string | string[]] }>()

interface Pending {
  id: number
  name: string
  phase: 'compress' | 'upload'
}

const input = ref<HTMLInputElement | null>(null)
const pending = ref<Pending[]>([])
const dragOver = ref(false)
let seq = 0

const list = computed<string[]>(() => {
  const v = props.modelValue
  if (Array.isArray(v)) return v
  return v ? [v] : []
})
const single = computed(() => (Array.isArray(props.modelValue) ? (props.modelValue[0] ?? '') : props.modelValue))
const remaining = computed(() => Math.max(0, props.max - list.value.length - pending.value.length))
const busy = computed(() => pending.value.length > 0)

function pick() {
  if (!props.disabled) input.value?.click()
}

function currentList(): string[] {
  const v = props.modelValue
  return Array.isArray(v) ? [...v] : v ? [v] : []
}

async function uploadOne(file: File): Promise<string | null> {
  const p: Pending = { id: ++seq, name: file.name, phase: 'compress' }
  pending.value.push(p)
  const set = (phase: Pending['phase']) => {
    const item = pending.value.find((x) => x.id === p.id)
    if (item) item.phase = phase
  }
  try {
    const compressed = await compressImage(file)
    set('upload')
    const res = await api.upload<{ url: string }>('/uploads', compressed, { kind: 'image' })
    return res.url
  } catch (e) {
    toast.error(`Gagal mengunggah ${file.name}: ${errorMessage(e)}`)
    return null
  } finally {
    pending.value = pending.value.filter((x) => x.id !== p.id)
  }
}

async function handleFiles(files: FileList | File[] | null) {
  if (!files || props.disabled) return
  let arr = Array.from(files).filter((f) => {
    if (!f.type.startsWith('image/')) {
      toast.error(`${f.name} bukan file gambar`)
      return false
    }
    if (f.size > props.maxSizeMb * 1024 * 1024) {
      toast.error(`${f.name} terlalu besar (maks. ${props.maxSizeMb} MB)`)
      return false
    }
    return true
  })
  if (!arr.length) return

  if (!props.multiple) {
    const url = await uploadOne(arr[0]!)
    if (url) emit('update:modelValue', Array.isArray(props.modelValue) ? [url] : url)
    return
  }

  if (arr.length > remaining.value) {
    toast.info(`Maksimal ${props.max} foto. Hanya ${remaining.value} foto pertama yang diunggah.`)
    arr = arr.slice(0, remaining.value)
  }
  for (const f of arr) {
    const url = await uploadOne(f)
    if (url) emit('update:modelValue', [...currentList(), url])
  }
}

function onChange(e: Event) {
  const el = e.target as HTMLInputElement
  const files = el.files ? Array.from(el.files) : []
  el.value = ''
  handleFiles(files)
}

function onDrop(e: DragEvent) {
  dragOver.value = false
  handleFiles(e.dataTransfer?.files ?? null)
}

function removeAt(i: number) {
  if (!props.multiple) {
    emit('update:modelValue', Array.isArray(props.modelValue) ? [] : '')
    return
  }
  const next = currentList()
  next.splice(i, 1)
  emit('update:modelValue', next)
}

function move(i: number, dir: -1 | 1) {
  const next = currentList()
  const j = i + dir
  if (j < 0 || j >= next.length) return
  ;[next[i], next[j]] = [next[j]!, next[i]!]
  emit('update:modelValue', next)
}
</script>

<template>
  <div>
    <input ref="input" type="file" accept="image/*" class="hidden" :multiple="multiple" @change="onChange" />

    <!-- Satu gambar -->
    <div v-if="!multiple" class="flex items-start gap-3">
      <div
        class="relative w-32 shrink-0 overflow-hidden rounded-xl border bg-surface-2 sm:w-40"
        :class="dragOver ? 'border-accent' : 'border-line'"
        :style="{ aspectRatio: aspect }"
        @dragover.prevent="dragOver = true"
        @dragleave="dragOver = false"
        @drop.prevent="onDrop"
      >
        <img v-if="single && !busy" :src="single" alt="" class="h-full w-full object-cover" loading="lazy" />
        <button
          v-else-if="!busy"
          type="button"
          class="flex h-full w-full flex-col items-center justify-center gap-1 p-2 text-center text-xs text-muted transition hover:bg-surface hover:text-ink"
          :disabled="disabled"
          @click="pick"
        >
          <UiIcon name="image" :size="22" />
          <span>Pilih foto</span>
        </button>
        <div v-if="busy" class="absolute inset-0 flex flex-col items-center justify-center gap-1.5 bg-surface-2 text-xs text-muted">
          <UiSpinner :size="20" />
          <span>{{ pending[0]?.phase === 'compress' ? 'Mengompres…' : 'Mengunggah…' }}</span>
        </div>
      </div>
      <div class="flex flex-col gap-1.5 pt-1">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg border border-line bg-surface px-3 py-1.5 text-sm text-ink transition hover:bg-surface-2 disabled:opacity-50"
          :disabled="busy || disabled"
          @click="pick"
        >
          <UiIcon name="upload" :size="15" /> {{ single ? 'Ganti' : 'Unggah' }}
        </button>
        <button
          v-if="single"
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm text-danger transition hover:bg-danger/10 disabled:opacity-50"
          :disabled="busy || disabled"
          @click="removeAt(0)"
        >
          <UiIcon name="trash" :size="15" /> Hapus
        </button>
        <p class="text-xs text-muted">JPG/PNG/WebP, dikompres otomatis.</p>
      </div>
    </div>

    <!-- Banyak gambar -->
    <div v-else>
      <div
        class="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-5"
        @dragover.prevent="dragOver = true"
        @dragleave="dragOver = false"
        @drop.prevent="onDrop"
      >
        <div
          v-for="(url, i) in list"
          :key="url + i"
          class="group relative overflow-hidden rounded-lg border border-line bg-surface-2"
          :style="{ aspectRatio: aspect }"
        >
          <img :src="url" alt="" class="h-full w-full object-cover" loading="lazy" />
          <span class="absolute top-1 left-1 rounded-md bg-ink/60 px-1.5 text-[11px] font-medium text-surface">{{ i + 1 }}</span>
          <div class="absolute inset-x-0 bottom-0 flex items-center justify-between gap-1 bg-gradient-to-t from-ink/70 to-transparent p-1 pt-4">
            <div class="flex gap-0.5">
              <button
                type="button"
                class="rounded-md bg-surface/90 p-1 text-ink disabled:opacity-40"
                :disabled="i === 0 || disabled"
                aria-label="Geser ke kiri"
                @click="move(i, -1)"
              >
                <UiIcon name="chevron-left" :size="14" />
              </button>
              <button
                type="button"
                class="rounded-md bg-surface/90 p-1 text-ink disabled:opacity-40"
                :disabled="i === list.length - 1 || disabled"
                aria-label="Geser ke kanan"
                @click="move(i, 1)"
              >
                <UiIcon name="chevron-right" :size="14" />
              </button>
            </div>
            <button
              type="button"
              class="rounded-md bg-surface/90 p-1 text-danger disabled:opacity-40"
              :disabled="disabled"
              aria-label="Hapus foto"
              @click="removeAt(i)"
            >
              <UiIcon name="trash" :size="14" />
            </button>
          </div>
        </div>

        <div
          v-for="p in pending"
          :key="'p' + p.id"
          class="flex flex-col items-center justify-center gap-1.5 rounded-lg border border-dashed border-line bg-surface-2 p-1 text-center text-[11px] text-muted"
          :style="{ aspectRatio: aspect }"
        >
          <UiSpinner :size="18" />
          <span>{{ p.phase === 'compress' ? 'Mengompres…' : 'Mengunggah…' }}</span>
        </div>

        <button
          v-if="remaining > 0"
          type="button"
          class="flex flex-col items-center justify-center gap-1 rounded-lg border border-dashed p-1 text-center text-xs text-muted transition hover:border-accent hover:text-accent disabled:opacity-50"
          :class="dragOver ? 'border-accent bg-accent/5' : 'border-line bg-surface'"
          :style="{ aspectRatio: aspect }"
          :disabled="disabled"
          @click="pick"
        >
          <UiIcon name="plus" :size="20" />
          <span>Tambah foto</span>
        </button>
      </div>
      <p class="mt-2 text-xs text-muted">{{ list.length }} / {{ max }} foto · bisa pilih beberapa sekaligus · dikompres otomatis</p>
    </div>
  </div>
</template>
