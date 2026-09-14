<script setup lang="ts">
import { computed } from 'vue'
import UiIcon from './UiIcon.vue'

const page = defineModel<number>('page', { default: 1 })
const props = defineProps<{ total: number; perPage: number }>()

const pages = computed(() => Math.max(1, Math.ceil(props.total / Math.max(1, props.perPage))))
const from = computed(() => (props.total === 0 ? 0 : (page.value - 1) * props.perPage + 1))
const to = computed(() => Math.min(props.total, page.value * props.perPage))
</script>

<template>
  <div v-if="total > 0" class="flex items-center justify-between gap-3 text-sm text-muted">
    <span>{{ from }}–{{ to }} dari {{ total }}</span>
    <div v-if="pages > 1" class="flex items-center gap-1">
      <button
        type="button"
        class="rounded-lg border border-line bg-surface p-1.5 text-ink transition hover:bg-surface-2 disabled:opacity-40"
        :disabled="page <= 1"
        aria-label="Halaman sebelumnya"
        @click="page = page - 1"
      >
        <UiIcon name="chevron-left" :size="16" />
      </button>
      <span class="px-2 text-ink">{{ page }} / {{ pages }}</span>
      <button
        type="button"
        class="rounded-lg border border-line bg-surface p-1.5 text-ink transition hover:bg-surface-2 disabled:opacity-40"
        :disabled="page >= pages"
        aria-label="Halaman berikutnya"
        @click="page = page + 1"
      >
        <UiIcon name="chevron-right" :size="16" />
      </button>
    </div>
  </div>
</template>
