<script setup lang="ts">
import { dismiss, toastState } from '../../composables/useToast'
import UiIcon from './UiIcon.vue'

const ICON = { success: 'check-circle', error: 'x-circle', info: 'info' } as const
const TONE = { success: 'text-success', error: 'text-danger', info: 'text-accent' } as const
</script>

<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed inset-x-0 top-0 z-[80] flex flex-col items-center gap-2 p-3 sm:inset-x-auto sm:right-0 sm:items-end sm:p-5"
      aria-live="polite"
    >
      <TransitionGroup
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="-translate-y-2 opacity-0"
        leave-active-class="transition duration-150 ease-in"
        leave-to-class="opacity-0"
      >
        <div
          v-for="t in toastState.items"
          :key="t.id"
          class="pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-xl border border-line bg-surface px-4 py-3 text-sm text-ink shadow-lg"
          role="status"
        >
          <UiIcon :name="ICON[t.kind]" :size="18" class="mt-0.5" :class="TONE[t.kind]" />
          <p class="min-w-0 flex-1 break-words whitespace-pre-line">{{ t.message }}</p>
          <button type="button" class="-m-1 rounded p-1 text-muted hover:text-ink" aria-label="Tutup" @click="dismiss(t.id)">
            <UiIcon name="x" :size="16" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
