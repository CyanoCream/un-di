<script setup lang="ts" generic="T extends string | number">
const model = defineModel<T>()

const props = withDefaults(
  defineProps<{
    type?: string
    placeholder?: string
    disabled?: boolean
    readonly?: boolean
    invalid?: boolean
    size?: 'sm' | 'md'
  }>(),
  { type: 'text', placeholder: undefined, disabled: false, readonly: false, invalid: false, size: 'md' },
)

function onInput(e: Event) {
  const raw = (e.target as HTMLInputElement).value
  if (props.type === 'number' || typeof model.value === 'number') {
    const n = raw === '' ? 0 : Number(raw)
    model.value = (Number.isNaN(n) ? 0 : n) as T
  } else {
    model.value = raw as T
  }
}
</script>

<template>
  <input
    :type="type"
    :value="model"
    :placeholder="placeholder"
    :disabled="disabled"
    :readonly="readonly"
    :aria-invalid="invalid || undefined"
    class="w-full min-w-0 rounded-xl border bg-surface-2 text-ink transition outline-none placeholder:text-muted/70 focus:border-accent focus:bg-surface focus:ring-3 focus:ring-accent/15 disabled:cursor-not-allowed disabled:opacity-60 read-only:bg-surface-2"
    :class="[invalid ? 'border-danger' : 'border-line', size === 'sm' ? 'h-8 px-2.5 text-sm' : 'h-10 px-3 text-sm']"
    @input="onInput"
  />
</template>
