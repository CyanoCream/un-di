<script setup lang="ts">
defineProps<{ title: string; subtitle?: string }>()
</script>

<template>
  <div class="mb-4 flex flex-wrap items-end justify-between gap-3">
    <div class="min-w-0">
      <slot name="eyebrow" />
      <h1 class="truncate text-lg font-semibold tracking-tight text-ink">{{ title }}</h1>
      <p v-if="subtitle" class="mt-0.5 text-xs text-muted">{{ subtitle }}</p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
