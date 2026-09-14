<script setup lang="ts">
import UiIcon from './UiIcon.vue'

const model = defineModel<number>({ default: 1 })

const props = withDefaults(defineProps<{ min?: number; max?: number; size?: 'md' | 'lg'; label?: string }>(), {
  min: 1,
  max: 99,
  size: 'md',
  label: 'Jumlah',
})

function set(v: number) {
  model.value = Math.min(props.max, Math.max(props.min, Math.round(v) || props.min))
}
</script>

<template>
  <div class="inline-flex items-center gap-2" role="group" :aria-label="label">
    <button
      type="button"
      class="flex items-center justify-center rounded-xl border border-line bg-surface text-ink transition hover:bg-surface-2 active:scale-95 disabled:opacity-40"
      :class="size === 'lg' ? 'size-14' : 'size-10'"
      :disabled="model <= min"
      aria-label="Kurangi"
      @click="set(model - 1)"
    >
      <UiIcon name="minus" :size="size === 'lg' ? 24 : 18" />
    </button>
    <output
      class="text-center font-semibold text-ink tabular-nums"
      :class="size === 'lg' ? 'min-w-14 text-3xl' : 'min-w-10 text-lg'"
      aria-live="polite"
    >
      {{ model }}
    </output>
    <button
      type="button"
      class="flex items-center justify-center rounded-xl border border-line bg-surface text-ink transition hover:bg-surface-2 active:scale-95 disabled:opacity-40"
      :class="size === 'lg' ? 'size-14' : 'size-10'"
      :disabled="model >= max"
      aria-label="Tambah"
      @click="set(model + 1)"
    >
      <UiIcon name="plus" :size="size === 'lg' ? 24 : 18" />
    </button>
  </div>
</template>
