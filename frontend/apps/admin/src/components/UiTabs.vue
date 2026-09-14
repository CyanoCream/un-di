<script setup lang="ts">
export interface TabItem {
  value: string
  label: string
  count?: number | null
}

defineProps<{ modelValue: string; tabs: TabItem[]; ariaLabel?: string }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

function onKey(e: KeyboardEvent, tabs: TabItem[], idx: number) {
  if (e.key !== 'ArrowRight' && e.key !== 'ArrowLeft') return
  e.preventDefault()
  const next = (idx + (e.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length
  emit('update:modelValue', tabs[next]!.value)
  const list = (e.currentTarget as HTMLElement).parentElement
  ;(list?.children[next] as HTMLElement | undefined)?.focus()
}
</script>

<template>
  <div class="-mx-px overflow-x-auto">
    <div role="tablist" :aria-label="ariaLabel" class="flex min-w-max gap-1 border-b border-line">
      <button
        v-for="(t, i) in tabs"
        :key="t.value"
        type="button"
        role="tab"
        :aria-selected="modelValue === t.value"
        :tabindex="modelValue === t.value ? 0 : -1"
        class="relative -mb-px inline-flex h-9 cursor-pointer items-center gap-2 border-b-2 px-3 text-sm font-medium whitespace-nowrap transition-colors"
        :class="modelValue === t.value ? 'border-accent text-ink' : 'border-transparent text-muted hover:text-ink'"
        @click="emit('update:modelValue', t.value)"
        @keydown="onKey($event, tabs, i)"
      >
        {{ t.label }}
        <span
          v-if="t.count !== undefined && t.count !== null"
          class="num rounded-full px-1.5 py-px text-[11px]"
          :class="modelValue === t.value ? 'bg-accent/15 text-accent' : 'bg-surface-2 text-muted'"
        >
          {{ t.count }}
        </span>
      </button>
    </div>
  </div>
</template>
