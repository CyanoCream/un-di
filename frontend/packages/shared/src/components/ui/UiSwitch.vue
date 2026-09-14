<script setup lang="ts">
const model = defineModel<boolean>({ default: false })

withDefaults(defineProps<{ label?: string; description?: string; disabled?: boolean }>(), {
  label: undefined,
  description: undefined,
  disabled: false,
})
</script>

<template>
  <label
    class="flex items-start gap-3 select-none"
    :class="disabled ? 'cursor-not-allowed opacity-60' : 'cursor-pointer'"
  >
    <button
      type="button"
      role="switch"
      :aria-checked="model"
      :disabled="disabled"
      class="relative mt-0.5 inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
      :class="model ? 'border-accent bg-accent' : 'border-line bg-surface-2'"
      @click="model = !model"
    >
      <span
        class="inline-block size-4.5 rounded-full shadow-sm transition-transform"
        :class="model ? 'translate-x-5.5 bg-accent-ink' : 'translate-x-0.5 bg-muted/60'"
      />
    </button>
    <span v-if="label || description || $slots.default" class="min-w-0">
      <span v-if="label" class="block text-sm font-medium text-ink">{{ label }}</span>
      <span v-if="description" class="block text-xs text-muted">{{ description }}</span>
      <slot />
    </span>
  </label>
</template>
