<script setup lang="ts">
import { computed } from 'vue'
import UiSpinner from './UiSpinner.vue'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
    size?: 'sm' | 'md'
    loading?: boolean
    disabled?: boolean
    type?: 'button' | 'submit' | 'reset'
    /** Jika diisi, dirender sebagai <a>. */
    href?: string
    block?: boolean
  }>(),
  { variant: 'primary', size: 'md', loading: false, disabled: false, type: 'button', href: undefined, block: false },
)

const VARIANTS = {
  primary: 'bg-accent text-accent-ink shadow-sm hover:brightness-95 active:brightness-90',
  secondary: 'border border-line bg-surface text-ink hover:bg-surface-2',
  ghost: 'text-ink hover:bg-surface-2',
  danger: 'border border-danger/30 bg-danger/10 text-danger hover:bg-danger/15',
} as const

const SIZES = {
  sm: 'h-8 px-3 text-sm gap-1.5 rounded-lg',
  md: 'h-10 px-4 text-sm gap-2 rounded-xl',
} as const

const cls = computed(() => [
  'inline-flex select-none items-center justify-center font-medium whitespace-nowrap transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:cursor-not-allowed disabled:opacity-50',
  VARIANTS[props.variant],
  SIZES[props.size],
  props.block ? 'w-full' : '',
  props.href && (props.disabled || props.loading) ? 'pointer-events-none opacity-50' : '',
])
</script>

<template>
  <a v-if="href" :href="href" :class="cls" :aria-disabled="disabled || loading || undefined">
    <UiSpinner v-if="loading" :size="size === 'sm' ? 14 : 16" />
    <slot />
  </a>
  <button v-else :type="type" :class="cls" :disabled="disabled || loading" :aria-busy="loading || undefined">
    <UiSpinner v-if="loading" :size="size === 'sm' ? 14 : 16" />
    <slot />
  </button>
</template>
