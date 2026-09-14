<script setup lang="ts">
import type { Invitation } from '@undangan/shared'
import AttendanceReport from '@undangan/shared/components/AttendanceReport.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import { computed } from 'vue'
import { useInvitationCtx } from '../../lib/invitation'

const ctx = useInvitationCtx()
const inv = computed(() => ctx.invitation.value as Invitation)
const stationPath = computed(() => inv.value.settings?.checkin_station_path || `/checkin/${inv.value.id}`)
</script>

<template>
  <div class="pb-6">
    <div class="mb-5 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h2 class="heading text-3xl">Kehadiran tamu</h2>
        <p class="mt-1 text-sm text-muted">Pantau siapa saja yang sudah datang di hari acara, lengkap dengan data RSVP.</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <UiButton v-if="inv.settings?.checkin_enabled" :href="stationPath" target="_blank" rel="noopener">
          <UiIcon name="qr" :size="16" /> Buka stasiun check-in
        </UiButton>
        <UiButton v-else variant="secondary" :href="`/undangan/${inv.id}/bagikan`" @click.prevent="$router.push(`/undangan/${inv.id}/bagikan`)">
          <UiIcon name="qr" :size="16" /> Atur check-in
        </UiButton>
      </div>
    </div>
    <AttendanceReport :invitation="inv" />
  </div>
</template>
