<script setup lang="ts">
import { APP_NAME } from '../lib/brand'

withDefaults(defineProps<{ size?: 'sm' | 'md' }>(), { size: 'md' })
</script>

<template>
  <span class="inline-flex items-center gap-2">
    <span
      class="flex items-center justify-center rounded-xl bg-accent text-accent-ink shadow-soft"
      :class="size === 'sm' ? 'size-8' : 'size-9'"
    >
      <svg :width="size === 'sm' ? 16 : 18" :height="size === 'sm' ? 16 : 18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <path d="M12 20.5s-7.5-4.6-7.5-10A4.2 4.2 0 0 1 12 8a4.2 4.2 0 0 1 7.5 2.5c0 5.4-7.5 10-7.5 10Z" />
      </svg>
    </span>
    <span class="leading-none">
      <span class="block font-display font-semibold text-ink" :class="size === 'sm' ? 'text-lg' : 'text-xl'">{{ APP_NAME }}</span>
      <span class="block text-[10px] tracking-[0.2em] text-muted uppercase">Portal Pelanggan</span>
    </span>
  </span>
</template>
