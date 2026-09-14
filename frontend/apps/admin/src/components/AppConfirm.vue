<script setup lang="ts">
import { computed } from 'vue'
import { pendingConfirm, settleConfirm } from '../lib/confirm'
import UiDialog from './UiDialog.vue'

const c = computed(() => pendingConfirm.value)
const btnClass = computed(() => ({ primary: 'btn-primary', danger: 'btn-danger', success: 'btn-success' })[c.value?.tone ?? 'primary'])
</script>

<template>
  <UiDialog :open="!!c" :title="c?.title" size="sm" @close="settleConfirm(false)">
    <div data-overlay-top class="space-y-3 text-sm">
      <p v-if="c?.message" class="text-ink/90">{{ c.message }}</p>
      <ul v-if="c?.details?.length" class="space-y-1.5 rounded-md border border-line bg-canvas/60 px-3 py-2.5 text-xs text-muted">
        <li v-for="(d, i) in c.details" :key="i" class="flex gap-2">
          <span class="mt-1.5 size-1 shrink-0 rounded-full bg-accent" />
          <span>{{ d }}</span>
        </li>
      </ul>
    </div>
    <template #footer>
      <button type="button" class="btn btn-secondary" @click="settleConfirm(false)">{{ c?.cancelText ?? 'Batal' }}</button>
      <button type="button" class="btn" :class="btnClass" data-autofocus @click="settleConfirm(true)">{{ c?.confirmText ?? 'Lanjutkan' }}</button>
    </template>
  </UiDialog>
</template>
