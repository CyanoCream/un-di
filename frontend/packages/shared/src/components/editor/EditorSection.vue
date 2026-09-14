<script setup lang="ts">
import UiIcon, { type IconName } from '../ui/UiIcon.vue'

defineProps<{ id: string; title: string; icon: IconName; open: boolean; summary?: string; done?: boolean }>()
defineEmits<{ toggle: [] }>()
</script>

<template>
  <section :id="id" class="scroll-mt-[calc(var(--editor-nav-top,0px)+4rem)] overflow-hidden rounded-2xl border border-line bg-surface">
    <button
      type="button"
      class="flex w-full items-center gap-3 px-4 py-3.5 text-left transition hover:bg-surface-2 sm:px-5"
      :aria-expanded="open"
      @click="$emit('toggle')"
    >
      <span
        class="flex size-8 shrink-0 items-center justify-center rounded-lg"
        :class="done ? 'bg-success/12 text-success' : 'bg-accent/10 text-accent'"
      >
        <UiIcon :name="done ? 'check' : icon" :size="16" />
      </span>
      <span class="min-w-0 flex-1">
        <span class="block text-[15px] font-semibold text-ink">{{ title }}</span>
        <span v-if="summary" class="block truncate text-xs text-muted">{{ summary }}</span>
      </span>
      <UiIcon name="chevron-down" :size="18" class="text-muted transition-transform" :class="open ? 'rotate-180' : ''" />
    </button>
    <div v-if="open" class="border-t border-line px-4 py-5 sm:px-5">
      <slot />
    </div>
  </section>
</template>
