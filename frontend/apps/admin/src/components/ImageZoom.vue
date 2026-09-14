<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ src: string | null; alt?: string }>()
const emit = defineEmits<{ close: [] }>()

const actual = ref(false)
const root = ref<HTMLElement | null>(null)

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.src) {
    e.stopImmediatePropagation()
    emit('close')
  }
}

watch(
  () => props.src,
  async (src) => {
    actual.value = false
    if (src) {
      window.addEventListener('keydown', onKey, true)
      await nextTick()
      root.value?.focus()
    } else {
      window.removeEventListener('keydown', onKey, true)
    }
  },
)
onBeforeUnmount(() => window.removeEventListener('keydown', onKey, true))
</script>

<template>
  <Teleport to="body">
    <Transition enter-active-class="transition-opacity duration-150" enter-from-class="opacity-0" leave-active-class="transition-opacity duration-100" leave-to-class="opacity-0">
      <div
        v-if="src"
        ref="root"
        data-overlay-top
        role="dialog"
        aria-modal="true"
        :aria-label="alt ?? 'Pratinjau gambar'"
        tabindex="-1"
        class="fixed inset-0 z-[60] flex flex-col bg-black/90 outline-none"
      >
        <div class="flex items-center justify-between gap-2 px-4 py-3 text-sm text-ink">
          <span class="truncate text-muted">{{ alt }}</span>
          <div class="flex items-center gap-1">
            <button type="button" class="btn btn-ghost btn-sm" @click="actual = !actual">
              <AppIcon name="zoom" :size="14" /> {{ actual ? 'Sesuaikan layar' : 'Ukuran asli' }}
            </button>
            <a :href="src" target="_blank" rel="noopener" class="btn btn-ghost btn-sm"><AppIcon name="external" :size="14" /> Tab baru</a>
            <button type="button" class="btn btn-ghost btn-sm btn-icon" aria-label="Tutup" @click="emit('close')"><AppIcon name="x" /></button>
          </div>
        </div>
        <div class="min-h-0 flex-1 overflow-auto p-4" :class="!actual && 'flex items-center justify-center'" @click.self="emit('close')">
          <img
            :src="src"
            :alt="alt"
            class="mx-auto rounded"
            :class="actual ? 'max-w-none cursor-zoom-out' : 'max-h-full max-w-full cursor-zoom-in object-contain'"
            @click="actual = !actual"
          />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
