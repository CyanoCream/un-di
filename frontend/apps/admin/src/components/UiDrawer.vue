<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import AppIcon from './AppIcon.vue'

const props = withDefaults(defineProps<{ open: boolean; title?: string; width?: string }>(), { width: 'max-w-xl' })
const emit = defineEmits<{ close: [] }>()

const panel = ref<HTMLElement | null>(null)
const titleId = useId()
let lastFocus: HTMLElement | null = null

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.open) {
    // Dialog/overlay lain di atas drawer menangani Esc sendiri.
    if (document.querySelector('[data-overlay-top]')) return
    emit('close')
  }
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      lastFocus = document.activeElement as HTMLElement | null
      document.addEventListener('keydown', onKey)
      await nextTick()
      panel.value?.focus()
    } else {
      document.removeEventListener('keydown', onKey)
      lastFocus?.focus?.()
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>

<template>
  <Teleport to="body">
    <Transition enter-active-class="transition-opacity duration-150" enter-from-class="opacity-0" leave-active-class="transition-opacity duration-150" leave-to-class="opacity-0">
      <div v-if="open" class="fixed inset-0 z-40 bg-black/50" @click="emit('close')" />
    </Transition>
    <Transition
      enter-active-class="transition-transform duration-200 ease-out"
      enter-from-class="translate-x-full"
      leave-active-class="transition-transform duration-150 ease-in"
      leave-to-class="translate-x-full"
    >
      <aside
        v-if="open"
        ref="panel"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        tabindex="-1"
        class="fixed inset-y-0 right-0 z-40 flex w-full flex-col border-l border-line bg-surface shadow-2xl shadow-black/60 outline-none"
        :class="width"
      >
        <header class="flex items-center justify-between gap-3 border-b border-line px-5 py-3">
          <div :id="titleId" class="min-w-0 flex-1">
            <slot name="header">
              <h2 class="text-[15px] font-semibold">{{ title }}</h2>
            </slot>
          </div>
          <button type="button" class="btn btn-ghost btn-icon h-7 w-7" aria-label="Tutup panel" @click="emit('close')">
            <AppIcon name="x" />
          </button>
        </header>
        <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
          <slot />
        </div>
        <footer v-if="$slots.footer" class="flex flex-wrap items-center justify-end gap-2 border-t border-line bg-surface-2/40 px-5 py-3">
          <slot name="footer" />
        </footer>
      </aside>
    </Transition>
  </Teleport>
</template>
