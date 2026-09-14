<script setup lang="ts">
withDefaults(defineProps<{ tone?: 'neutral' | 'accent' | 'success' | 'warning' | 'danger' }>(), { tone: 'neutral' })

const TONES = {
  neutral: 'bg-surface-2 text-muted border-line',
  accent: 'bg-accent/12 text-accent border-accent/25',
  success: 'bg-success/12 text-success border-success/25',
  warning: 'bg-warning/15 text-warning border-warning/30',
  danger: 'bg-danger/12 text-danger border-danger/25',
} as const
</script>

<template>
  <span
    class="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium whitespace-nowrap"
    :class="TONES[tone]"
  >
    <slot />
  </span>
</template>
