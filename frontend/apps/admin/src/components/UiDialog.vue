<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import AppIcon from './AppIcon.vue'

const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    description?: string
    size?: 'sm' | 'md' | 'lg' | 'xl' | 'full'
    /** Cegah tutup via Esc/klik luar (mis. saat proses simpan). */
    persistent?: boolean
  }>(),
  { size: 'md', persistent: false },
)
const emit = defineEmits<{ close: [] }>()

const panel = ref<HTMLElement | null>(null)
const titleId = useId()
let lastFocus: HTMLElement | null = null
let wasOpen = false

const WIDTH = { sm: 'max-w-sm', md: 'max-w-lg', lg: 'max-w-2xl', xl: 'max-w-4xl', full: 'max-w-6xl' }

function focusables(): HTMLElement[] {
  if (!panel.value) return []
  return Array.from(
    panel.value.querySelectorAll<HTMLElement>(
      'a[href],button:not([disabled]),input:not([disabled]),select:not([disabled]),textarea:not([disabled]),iframe,[tabindex]:not([tabindex="-1"])',
    ),
  ).filter((el) => el.offsetParent !== null || el === document.activeElement)
}

function onKey(e: KeyboardEvent) {
  if (!props.open) return
  if (e.key === 'Escape') {
    e.stopPropagation()
    if (!props.persistent) emit('close')
  } else if (e.key === 'Tab') {
    const els = focusables()
    if (!els.length) return
    const first = els[0]!
    const last = els[els.length - 1]!
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      wasOpen = true
      lastFocus = document.activeElement as HTMLElement | null
      document.body.style.overflow = 'hidden'
      await nextTick()
      const auto = panel.value?.querySelector<HTMLElement>('[autofocus],[data-autofocus]')
      ;(auto ?? focusables()[0] ?? panel.value)?.focus()
    } else if (wasOpen) {
      wasOpen = false
      document.body.style.overflow = ''
      lastFocus?.focus?.()
      lastFocus = null
    }
  },
  { immediate: true },
)
onBeforeUnmount(() => {
  if (props.open) document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0"
      leave-active-class="transition duration-100 ease-in"
      leave-to-class="opacity-0"
    >
      <div
        v-if="open"
        data-overlay-top
        class="fixed inset-0 z-50 flex items-end justify-center overflow-y-auto bg-black/60 p-0 backdrop-blur-[2px] sm:items-start sm:p-6 sm:pt-[8vh]"
        @mousedown.self="!persistent && emit('close')"
        @keydown="onKey"
      >
        <div
          ref="panel"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="title ? titleId : undefined"
          tabindex="-1"
          class="flex max-h-[92dvh] w-full flex-col rounded-t-xl border border-line bg-surface shadow-2xl shadow-black/50 outline-none sm:max-h-[84vh] sm:rounded-xl"
          :class="WIDTH[size]"
        >
          <header v-if="title || $slots.header" class="flex items-start justify-between gap-4 border-b border-line px-5 py-3.5">
            <slot name="header">
              <div class="min-w-0">
                <h2 :id="titleId" class="text-[15px] font-semibold text-ink">{{ title }}</h2>
                <p v-if="description" class="mt-0.5 text-xs text-muted">{{ description }}</p>
              </div>
            </slot>
            <button type="button" class="btn btn-ghost btn-icon -mr-2 h-7 w-7" aria-label="Tutup" :disabled="persistent" @click="emit('close')">
              <AppIcon name="x" />
            </button>
          </header>
          <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
            <slot />
          </div>
          <footer v-if="$slots.footer" class="flex flex-wrap items-center justify-end gap-2 border-t border-line bg-surface-2/40 px-5 py-3">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
