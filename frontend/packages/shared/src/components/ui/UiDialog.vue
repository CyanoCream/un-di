<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import UiIcon from './UiIcon.vue'

const open = defineModel<boolean>('open', { default: false })

const props = withDefaults(
  defineProps<{
    title?: string
    description?: string
    size?: 'sm' | 'md' | 'lg' | 'full'
    /** Nonaktifkan tutup via Esc / klik latar (mis. saat proses berjalan). */
    persistent?: boolean
    /** Hilangkan padding isi (mis. untuk iframe). */
    flush?: boolean
  }>(),
  { title: undefined, description: undefined, size: 'md', persistent: false, flush: false },
)

const emit = defineEmits<{ close: [] }>()

const SIZES = {
  sm: 'sm:max-w-md',
  md: 'sm:max-w-xl',
  lg: 'sm:max-w-3xl',
  full: '',
} as const

// ---- Tumpukan dialog global: Esc hanya menutup dialog paling atas; scroll lock berbasis hitungan. ----
const uid = Symbol('dialog')
const panel = ref<HTMLElement | null>(null)
let lastFocus: HTMLElement | null = null

function close() {
  if (props.persistent) return
  open.value = false
  emit('close')
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && stack[stack.length - 1] === uid) {
    e.stopPropagation()
    close()
  }
}

function activate() {
  stack.push(uid)
  if (stack.length === 1) {
    scrollbarGap = window.innerWidth - document.documentElement.clientWidth
    document.body.style.overflow = 'hidden'
    if (scrollbarGap > 0) document.body.style.paddingRight = `${scrollbarGap}px`
  }
  lastFocus = document.activeElement as HTMLElement | null
  document.addEventListener('keydown', onKey)
  nextTick(() => {
    // Di perangkat sentuh jangan fokus ke input (keyboard virtual langsung muncul).
    const fine = window.matchMedia?.('(pointer: fine)').matches
    const selector = fine ? '[autofocus], input:not([type=hidden]):not([type=file]), textarea, select' : '[autofocus]'
    const target = panel.value?.querySelector<HTMLElement>(selector)
    ;(target ?? panel.value)?.focus({ preventScroll: true })
  })
}

function deactivate() {
  const i = stack.indexOf(uid)
  if (i < 0) return
  stack.splice(i, 1)
  document.removeEventListener('keydown', onKey)
  if (stack.length === 0) {
    document.body.style.overflow = ''
    document.body.style.paddingRight = ''
  }
  lastFocus?.focus?.({ preventScroll: true })
}

watch(
  open,
  (v) => {
    if (v) activate()
    else deactivate()
  },
  { immediate: true },
)
onBeforeUnmount(deactivate)
</script>

<script lang="ts">
const stack: symbol[] = []
let scrollbarGap = 0
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      leave-active-class="transition duration-150 ease-in"
      leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 z-[60] flex" :class="size === 'full' ? 'sm:p-4' : 'items-end justify-center sm:items-center sm:p-6'">
        <div class="absolute inset-0 bg-ink/45 backdrop-blur-[2px]" @click="close" />
        <div
          ref="panel"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          tabindex="-1"
          class="relative flex w-full flex-col bg-surface text-ink shadow-2xl outline-none"
          :class="[
            size === 'full'
              ? 'h-full sm:rounded-2xl'
              : 'max-h-[92dvh] rounded-t-2xl sm:max-h-[88dvh] sm:rounded-2xl',
            SIZES[size],
          ]"
        >
          <div v-if="title || $slots.header" class="flex items-start gap-3 border-b border-line px-5 py-4">
            <div class="min-w-0 flex-1">
              <slot name="header">
                <h2 class="text-base font-semibold text-ink">{{ title }}</h2>
                <p v-if="description" class="mt-0.5 text-sm text-muted">{{ description }}</p>
              </slot>
            </div>
            <slot name="actions" />
            <button
              v-if="!persistent"
              type="button"
              class="-m-1.5 rounded-lg p-1.5 text-muted transition hover:bg-surface-2 hover:text-ink"
              aria-label="Tutup"
              @click="close"
            >
              <UiIcon name="x" :size="20" />
            </button>
          </div>
          <div class="min-h-0 flex-1 overflow-y-auto overscroll-contain" :class="flush ? '' : 'px-5 py-4'">
            <slot />
          </div>
          <div
            v-if="$slots.footer"
            class="flex flex-wrap items-center justify-end gap-2 border-t border-line px-5 py-3 pb-[max(0.75rem,env(safe-area-inset-bottom))]"
          >
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
