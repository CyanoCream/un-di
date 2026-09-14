<script setup lang="ts">
import AppIcon, { type IconName } from './AppIcon.vue'
import UiSpinner from './UiSpinner.vue'

/**
 * Pembungkus status data: loading → error → kosong → isi (slot default).
 * `loading` hanya menutupi isi bila belum ada data (`empty` true) supaya reload tidak berkedip.
 */
withDefaults(
  defineProps<{
    loading?: boolean
    error?: string
    empty?: boolean
    emptyTitle?: string
    emptyText?: string
    emptyIcon?: IconName
    compact?: boolean
  }>(),
  { emptyTitle: 'Belum ada data', emptyIcon: 'layers' },
)
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div v-if="loading && empty" class="flex items-center justify-center gap-2 text-sm text-muted" :class="compact ? 'py-6' : 'py-16'" role="status">
    <UiSpinner /> Memuat…
  </div>
  <div v-else-if="error" class="flex flex-col items-center justify-center gap-3 text-center" :class="compact ? 'py-6' : 'py-14'" role="alert">
    <span class="grid size-9 place-items-center rounded-full bg-danger/10 text-danger"><AppIcon name="alert" :size="18" /></span>
    <div>
      <p class="text-sm font-medium text-ink">Gagal memuat data</p>
      <p class="mt-0.5 text-xs text-muted">{{ error }}</p>
    </div>
    <button type="button" class="btn btn-secondary btn-sm" @click="emit('retry')"><AppIcon name="refresh" :size="14" /> Coba lagi</button>
  </div>
  <div v-else-if="empty" class="flex flex-col items-center justify-center gap-2 text-center" :class="compact ? 'py-6' : 'py-14'">
    <span class="grid size-9 place-items-center rounded-full bg-surface-2 text-muted"><AppIcon :name="emptyIcon" :size="18" /></span>
    <p class="text-sm font-medium text-ink">{{ emptyTitle }}</p>
    <p v-if="emptyText" class="max-w-sm text-xs text-muted">{{ emptyText }}</p>
    <slot name="empty-action" />
  </div>
  <div v-else class="relative" :class="loading && 'pointer-events-none opacity-60 transition-opacity'" :aria-busy="loading">
    <slot />
  </div>
</template>
