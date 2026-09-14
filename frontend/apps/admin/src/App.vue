<script setup lang="ts">
import UiConfirm from '@undangan/shared/components/ui/UiConfirm.vue'
import UiToaster from '@undangan/shared/components/ui/UiToaster.vue'
import AppConfirm from './components/AppConfirm.vue'
</script>

<template>
  <RouterView />
  <!-- Toast & konfirmasi dari komponen shared -->
  <UiToaster />
  <UiConfirm />
  <!-- Konfirmasi admin (dengan butir rincian dampak) -->
  <AppConfirm />
</template>
