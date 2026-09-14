<script setup lang="ts">
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import AuthShell from '../components/AuthShell.vue'
</script>

<template>
  <AuthShell title="Halaman tidak ditemukan" subtitle="Tautan yang Anda buka mungkin salah atau sudah tidak berlaku.">
    <UiButton block href="/" @click.prevent="$router.push('/')">Kembali ke beranda</UiButton>
  </AuthShell>
</template>
