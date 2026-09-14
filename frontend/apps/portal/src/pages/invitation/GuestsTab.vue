<script setup lang="ts">
import type { Invitation } from '@undangan/shared'
import GuestManager from '@undangan/shared/components/GuestManager.vue'
import { computed } from 'vue'
import { useInvitationCtx } from '../../lib/invitation'

const ctx = useInvitationCtx()
const inv = computed(() => ctx.invitation.value as Invitation)
</script>

<template>
  <div class="pb-6">
    <div class="mb-5">
      <h2 class="heading text-3xl">Daftar tamu</h2>
      <p class="mt-1 text-sm text-muted">Setiap tamu mendapat link personal dengan namanya di sampul undangan.</p>
    </div>
    <GuestManager :invitation="inv" @changed="ctx.reload()" />
  </div>
</template>
