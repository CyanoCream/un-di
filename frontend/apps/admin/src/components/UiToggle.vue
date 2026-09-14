<script setup lang="ts">
defineProps<{ modelValue: boolean; label?: string; disabled?: boolean; tone?: 'accent' | 'success' | 'warning' | 'danger' }>()
const emit = defineEmits<{ 'update:modelValue': [v: boolean] }>()
</script>

<template>
  <label class="inline-flex cursor-pointer items-center gap-2 select-none" :class="disabled && 'pointer-events-none opacity-50'">
    <button
      type="button"
      role="switch"
      :aria-checked="modelValue"
      :aria-label="label"
      :disabled="disabled"
      class="relative inline-flex h-5 w-9 shrink-0 items-center rounded-full ring-1 transition-colors ring-inset"
      :class="
        modelValue
          ? {
              accent: 'bg-accent ring-accent',
              success: 'bg-success ring-success',
              warning: 'bg-warning ring-warning',
              danger: 'bg-danger ring-danger',
            }[tone ?? 'accent']
          : 'bg-surface-2 ring-line'
      "
      @click="emit('update:modelValue', !modelValue)"
    >
      <span
        class="inline-block size-3.5 rounded-full shadow transition-transform"
        :class="modelValue ? 'translate-x-[18px] bg-canvas' : 'translate-x-[3px] bg-muted'"
      />
    </button>
    <span v-if="label || $slots.default" class="text-sm text-ink"><slot>{{ label }}</slot></span>
  </label>
</template>
