<script setup lang="ts">
import { computed } from 'vue'
import { formatNumber } from '../lib/util'
import AppIcon from './AppIcon.vue'

const props = defineProps<{ page: number; total: number; perPage: number }>()
const emit = defineEmits<{ 'update:page': [p: number] }>()

const pages = computed(() => Math.max(1, Math.ceil(props.total / props.perPage)))
const from = computed(() => (props.total === 0 ? 0 : (props.page - 1) * props.perPage + 1))
const to = computed(() => Math.min(props.total, props.page * props.perPage))
</script>

<template>
  <div class="flex flex-wrap items-center justify-between gap-2 border-t border-line px-3 py-2 text-xs text-muted">
    <span class="num">{{ formatNumber(from) }}–{{ formatNumber(to) }} dari {{ formatNumber(total) }}</span>
    <div v-if="pages > 1" class="flex items-center gap-1">
      <button type="button" class="btn btn-ghost btn-sm btn-icon" :disabled="page <= 1" aria-label="Halaman sebelumnya" @click="emit('update:page', page - 1)">
        <AppIcon name="chevron-left" />
      </button>
      <span class="num px-1">Hal. {{ page }} / {{ pages }}</span>
      <button type="button" class="btn btn-ghost btn-sm btn-icon" :disabled="page >= pages" aria-label="Halaman berikutnya" @click="emit('update:page', page + 1)">
        <AppIcon name="chevron-right" />
      </button>
    </div>
  </div>
</template>
