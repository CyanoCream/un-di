<script setup lang="ts" generic="T extends string | number">
const model = defineModel<T>()

const props = withDefaults(
  defineProps<{
    options: readonly { value: T; label: string }[]
    placeholder?: string
    disabled?: boolean
    invalid?: boolean
    size?: 'sm' | 'md'
  }>(),
  { placeholder: undefined, disabled: false, invalid: false, size: 'md' },
)

function onChange(e: Event) {
  const idx = (e.target as HTMLSelectElement).selectedIndex - (props.placeholder !== undefined ? 1 : 0)
  const opt = props.options[idx]
  if (opt) model.value = opt.value
}
</script>

<template>
  <div class="relative min-w-0">
    <select
      :disabled="disabled"
      :aria-invalid="invalid || undefined"
      class="w-full min-w-0 appearance-none rounded-xl border bg-surface-2 pr-9 text-ink transition outline-none focus:border-accent focus:bg-surface focus:ring-3 focus:ring-accent/15 disabled:opacity-60"
      :class="[invalid ? 'border-danger' : 'border-line', size === 'sm' ? 'h-8 pl-2.5 text-sm' : 'h-10 pl-3 text-sm']"
      @change="onChange"
    >
      <option v-if="placeholder !== undefined" value="" disabled :selected="model === undefined || model === ''">
        {{ placeholder }}
      </option>
      <option v-for="o in options" :key="String(o.value)" :value="o.value" :selected="o.value === model">
        {{ o.label }}
      </option>
    </select>
    <svg
      class="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-muted"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      aria-hidden="true"
    >
      <path d="m6 9 6 6 6-6" />
    </svg>
  </div>
</template>
