<script setup lang="ts">
import AppIcon from '../components/AppIcon.vue'
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-3 py-24 text-center">
    <span class="grid size-12 place-items-center rounded-full bg-surface-2 text-muted"><AppIcon name="alert" :size="22" /></span>
    <h1 class="text-lg font-semibold">Halaman tidak ditemukan</h1>
    <p class="text-sm text-muted">Alamat yang Anda buka tidak tersedia di Admin Console.</p>
    <RouterLink to="/" class="btn btn-secondary mt-2"><AppIcon name="dashboard" :size="14" /> Ke Dashboard</RouterLink>
  </div>
</template>
