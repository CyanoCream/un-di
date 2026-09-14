<script setup lang="ts">
import type { Invitation } from '@undangan/shared'
import WishesManager from '@undangan/shared/components/WishesManager.vue'
import { computed } from 'vue'
import { useInvitationCtx } from '../../lib/invitation'

const ctx = useInvitationCtx()
const inv = computed(() => ctx.invitation.value as Invitation)
</script>

<template>
  <div class="pb-6">
    <div class="mb-5">
      <h2 class="heading text-3xl">Ucapan & kehadiran</h2>
      <p class="mt-1 text-sm text-muted">
        Konfirmasi kehadiran dan doa dari tamu. Sembunyikan ucapan yang tidak pantas agar tidak tampil di undangan.
      </p>
    </div>
    <p v-if="!inv.content.rsvp.enabled" class="mb-4 rounded-2xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-ink">
      RSVP & ucapan sedang dinonaktifkan. Aktifkan di tab
      <RouterLink :to="`/undangan/${inv.id}/data`" class="font-medium text-accent hover:underline">Isi Data</RouterLink> → RSVP & Ucapan.
    </p>
    <WishesManager :invitation-id="inv.id" />
  </div>
</template>
