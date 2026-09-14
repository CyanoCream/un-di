<script setup lang="ts">
withDefaults(defineProps<{ label?: string; hint?: string; error?: string | null; required?: boolean }>(), {
  label: undefined,
  hint: undefined,
  error: undefined,
  required: false,
})
</script>

<template>
  <div class="min-w-0">
    <div v-if="label || $slots.aside" class="mb-1.5 flex items-baseline justify-between gap-2">
      <span v-if="label" class="text-sm font-medium text-ink">
        {{ label }}<span v-if="required" class="text-danger"> *</span>
      </span>
      <slot name="aside" />
    </div>
    <slot />
    <p v-if="error" class="mt-1 text-xs text-danger">{{ error }}</p>
    <p v-else-if="hint" class="mt-1 text-xs text-muted">{{ hint }}</p>
  </div>
</template>
