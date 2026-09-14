<script setup lang="ts">
import { computed } from 'vue'
import { confirmState, settleConfirm } from '../../composables/useConfirm'
import UiButton from './UiButton.vue'
import UiDialog from './UiDialog.vue'

const open = computed({
  get: () => confirmState.open,
  set: (v: boolean) => {
    if (!v) settleConfirm(false)
  },
})
</script>

<template>
  <UiDialog v-model:open="open" :title="confirmState.options.title" size="sm">
    <p class="text-sm leading-relaxed whitespace-pre-line text-muted">{{ confirmState.options.message }}</p>
    <template #footer>
      <UiButton variant="ghost" @click="settleConfirm(false)">{{ confirmState.options.cancelText }}</UiButton>
      <UiButton :variant="confirmState.options.danger ? 'danger' : 'primary'" autofocus @click="settleConfirm(true)">
        {{ confirmState.options.confirmText }}
      </UiButton>
    </template>
  </UiDialog>
</template>
