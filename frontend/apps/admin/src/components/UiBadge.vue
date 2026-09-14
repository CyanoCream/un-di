<script setup lang="ts">
import type { Tone } from '../lib/labels'

withDefaults(defineProps<{ tone?: Tone; dot?: boolean }>(), { tone: 'neutral', dot: true })

const TONES: Record<Tone, string> = {
  neutral: 'bg-muted/10 text-muted ring-muted/20',
  accent: 'bg-accent/10 text-accent ring-accent/25',
  success: 'bg-success/10 text-success ring-success/25',
  warning: 'bg-warning/10 text-warning ring-warning/25',
  danger: 'bg-danger/10 text-danger ring-danger/25',
}
</script>

<template>
  <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium whitespace-nowrap ring-1 ring-inset" :class="TONES[tone]">
    <span v-if="dot" class="size-1.5 rounded-full bg-current" aria-hidden="true" />
    <slot />
  </span>
</template>
