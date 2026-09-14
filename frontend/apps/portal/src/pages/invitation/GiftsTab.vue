<script setup lang="ts">
import type { Invitation } from '@undangan/shared'
import GiftConfirmations from '@undangan/shared/components/GiftConfirmations.vue'
import { computed } from 'vue'
import { useInvitationCtx } from '../../lib/invitation'

const ctx = useInvitationCtx()
const inv = computed(() => ctx.invitation.value as Invitation)
const confirmationOff = computed(() => !inv.value.content.gift.enabled || !inv.value.content.gift.confirmation_enabled)
</script>

<template>
  <div class="pb-6">
    <div class="mb-5">
      <h2 class="heading text-3xl">Konfirmasi hadiah</h2>
      <p class="mt-1 text-sm text-muted">Bukti transfer dan kado yang dikirim tamu dari halaman undangan. Foto bukti hanya bisa dilihat oleh Anda.</p>
    </div>
    <p v-if="confirmationOff" class="mb-4 rounded-2xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-ink">
      Konfirmasi hadiah sedang nonaktif. Aktifkan di tab
      <RouterLink :to="`/undangan/${inv.id}/data`" class="font-medium text-accent hover:underline">Isi Data</RouterLink> → Amplop Digital →
      “Izinkan tamu mengirim bukti transfer/kado”.
    </p>
    <GiftConfirmations :invitation-id="inv.id" />
  </div>
</template>
