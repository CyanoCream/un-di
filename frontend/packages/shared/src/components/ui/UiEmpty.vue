<script setup lang="ts">
import UiIcon, { type IconName } from './UiIcon.vue'

withDefaults(defineProps<{ title: string; description?: string; icon?: IconName; compact?: boolean }>(), {
  description: undefined,
  icon: 'sparkles',
  compact: false,
})
</script>

<template>
  <div class="flex flex-col items-center justify-center text-center" :class="compact ? 'px-4 py-8' : 'px-6 py-14'">
    <div class="mb-3 flex size-12 items-center justify-center rounded-full bg-surface-2 text-muted">
      <UiIcon :name="icon" :size="22" />
    </div>
    <p class="font-medium text-ink">{{ title }}</p>
    <p v-if="description" class="mt-1 max-w-sm text-sm text-muted">{{ description }}</p>
    <div v-if="$slots.default" class="mt-4 flex flex-wrap justify-center gap-2">
      <slot />
    </div>
  </div>
</template>
