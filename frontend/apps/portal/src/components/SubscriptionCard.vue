<script setup lang="ts">
import { daysUntil, formatDate, type Subscription } from '@undangan/shared'
import UiBadge from '@undangan/shared/components/ui/UiBadge.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import { computed } from 'vue'
import { SUB_STATUS } from '../lib/labels'

const props = defineProps<{ subscription: Subscription | null; invitationCount: number }>()

const active = computed(() => props.subscription && (props.subscription.status === 'active' || props.subscription.status === 'grace'))
const daysLeft = computed(() => daysUntil(props.subscription?.ends_at))
const totalDays = computed(() => {
  const s = props.subscription
  if (!s) return 1
  return Math.max(1, Math.round((new Date(s.ends_at).getTime() - new Date(s.starts_at).getTime()) / 86_400_000))
})
const pct = computed(() => Math.min(100, Math.max(0, (daysLeft.value / totalDays.value) * 100)))
</script>

<template>
  <div class="card relative overflow-hidden">
    <template v-if="subscription && active">
      <div
        v-if="subscription.status === 'grace'"
        class="flex items-start gap-2 border-b border-warning/25 bg-warning/10 px-5 py-3 text-sm text-ink"
      >
        <UiIcon name="alert" :size="17" class="mt-0.5 text-warning" />
        <p>
          Masa aktif habis, undangan akan dinonaktifkan pada
          <strong>{{ formatDate(subscription.grace_ends_at) }}</strong>. Perpanjang paket agar undangan tetap bisa diakses.
        </p>
      </div>
      <div class="p-5">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-xs tracking-wider text-muted uppercase">Paket Anda</p>
            <p class="heading mt-0.5 text-3xl">{{ subscription.plan_name }}</p>
          </div>
          <UiBadge :tone="SUB_STATUS[subscription.status].tone">{{ SUB_STATUS[subscription.status].label }}</UiBadge>
        </div>

        <div class="mt-5">
          <div class="flex items-baseline justify-between text-sm">
            <span class="text-muted">Sisa masa aktif</span>
            <span class="font-semibold text-ink">
              <template v-if="subscription.status === 'grace'">Habis</template>
              <template v-else>{{ daysLeft }} hari</template>
            </span>
          </div>
          <div class="mt-2 h-2 overflow-hidden rounded-full bg-surface-2">
            <div class="h-full rounded-full" :class="daysLeft <= 7 ? 'bg-warning' : 'bg-accent'" :style="{ width: `${subscription.status === 'grace' ? 0 : pct}%` }" />
          </div>
          <p class="mt-2 text-xs text-muted">Berakhir {{ formatDate(subscription.ends_at) }}</p>
        </div>

        <dl class="mt-5 grid grid-cols-2 gap-3 border-t border-line pt-4 text-sm">
          <div>
            <dt class="text-xs text-muted">Undangan</dt>
            <dd class="font-medium text-ink">{{ invitationCount }} / {{ subscription.max_invitations > 0 ? subscription.max_invitations : '∞' }}</dd>
          </div>
          <div>
            <dt class="text-xs text-muted">Maks. tamu / undangan</dt>
            <dd class="font-medium text-ink">{{ subscription.max_guests > 0 ? subscription.max_guests.toLocaleString('id-ID') : 'Tanpa batas' }}</dd>
          </div>
        </dl>

        <div class="mt-4 flex gap-2">
          <UiButton :variant="subscription.status === 'grace' || daysLeft <= 14 ? 'primary' : 'secondary'" size="sm" href="/paket" @click.prevent="$router.push('/paket')">
            {{ subscription.status === 'grace' || daysLeft <= 14 ? 'Perpanjang paket' : 'Lihat paket' }}
          </UiButton>
        </div>
      </div>
    </template>

    <div v-else class="relative p-6 text-center sm:p-8">
      <div class="mx-auto mb-3 flex size-12 items-center justify-center rounded-full bg-accent/10 text-accent">
        <UiIcon name="sparkles" :size="22" />
      </div>
      <p class="heading text-2xl">Belum ada paket aktif</p>
      <p class="mx-auto mt-1 max-w-sm text-sm text-muted">Pilih paket untuk mulai membuat undangan digital, mengundang tamu, dan menerima ucapan.</p>
      <UiButton class="mt-4" href="/paket" @click.prevent="$router.push('/paket')"><UiIcon name="gift" :size="16" /> Pilih paket</UiButton>
    </div>
  </div>
</template>
