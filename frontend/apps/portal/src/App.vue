<script setup lang="ts">
import UiConfirm from '@undangan/shared/components/ui/UiConfirm.vue'
import UiToaster from '@undangan/shared/components/ui/UiToaster.vue'
</script>

<template>
  <RouterView />
  <UiToaster />
  <UiConfirm />
</template>
