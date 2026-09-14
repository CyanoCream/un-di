<script setup lang="ts">
import { formatDate, type InvitationSummary } from '@undangan/shared'
import UiBadge from '@undangan/shared/components/ui/UiBadge.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import { INV_STATUS } from '../lib/labels'

defineProps<{ invitation: InvitationSummary }>()
</script>

<template>
  <RouterLink
    :to="`/undangan/${invitation.id}/data`"
    class="group card relative flex flex-col overflow-hidden p-5 transition hover:-translate-y-0.5 hover:shadow-lift"
  >
    <div class="pointer-events-none absolute -top-8 -right-8 size-28 rounded-full bg-accent/6" aria-hidden="true" />
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <p class="text-xs tracking-wider text-muted uppercase">Undangan</p>
        <h3 class="heading mt-0.5 truncate text-2xl">{{ invitation.title || '(belum diisi)' }}</h3>
      </div>
      <UiBadge :tone="INV_STATUS[invitation.status]?.tone ?? 'neutral'">{{ INV_STATUS[invitation.status]?.label ?? invitation.status }}</UiBadge>
    </div>
    <dl class="mt-4 grid grid-cols-2 gap-3 text-sm">
      <div>
        <dt class="flex items-center gap-1 text-xs text-muted"><UiIcon name="calendar" :size="13" /> Tanggal</dt>
        <dd class="mt-0.5 text-ink">{{ invitation.event_date ? formatDate(invitation.event_date) : 'Belum diatur' }}</dd>
      </div>
      <div>
        <dt class="flex items-center gap-1 text-xs text-muted"><UiIcon name="users" :size="13" /> Tamu</dt>
        <dd class="mt-0.5 text-ink">{{ invitation.guest_count }} tamu</dd>
      </div>
    </dl>
    <div class="mt-4 flex items-center justify-between gap-2 border-t border-line pt-3 text-xs">
      <span v-if="invitation.url" class="flex min-w-0 items-center gap-1 truncate text-muted">
        <UiIcon name="globe" :size="13" /> <span class="truncate">{{ invitation.url.replace(/^https?:\/\//, '') }}</span>
      </span>
      <span v-else class="flex items-center gap-1 text-warning"><UiIcon name="alert" :size="13" /> Subdomain belum diatur</span>
      <span class="flex shrink-0 items-center gap-0.5 font-medium text-accent">
        Kelola <UiIcon name="chevron-right" :size="14" class="transition group-hover:translate-x-0.5" />
      </span>
    </div>
  </RouterLink>
</template>
