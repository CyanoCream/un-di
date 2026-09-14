<script setup lang="ts">
import { errorMessage } from '@undangan/shared'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiEmpty from '@undangan/shared/components/ui/UiEmpty.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { onMounted, ref } from 'vue'
import CreateInvitationDialog from '../components/CreateInvitationDialog.vue'
import InvitationCard from '../components/InvitationCard.vue'
import PageHeader from '../components/PageHeader.vue'
import { useAccountState } from '../lib/account'

const { subscription, invitations, invitationTotal, loading, error, load, createBlockReason } = useAccountState()
const createOpen = ref(false)

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 pt-2 md:px-6 md:pt-10">
    <PageHeader
      title="Undangan"
      :subtitle="subscription && subscription.max_invitations > 0 ? `${invitationTotal} dari ${subscription.max_invitations} undangan terpakai` : 'Kelola semua undangan digital Anda'"
    >
      <UiButton :disabled="loading || !!createBlockReason" @click="createOpen = true">
        <UiIcon name="plus" :size="16" /> Buat Undangan
      </UiButton>
    </PageHeader>

    <div
      v-if="!loading && createBlockReason"
      class="mb-5 flex flex-col gap-3 rounded-2xl border border-warning/30 bg-warning/10 p-4 text-sm text-ink sm:flex-row sm:items-center"
    >
      <UiIcon name="info" :size="18" class="shrink-0 text-warning" />
      <p class="flex-1">{{ createBlockReason }}</p>
      <UiButton size="sm" href="/paket" @click.prevent="$router.push('/paket')">Lihat paket</UiButton>
    </div>

    <div v-if="loading" class="flex items-center justify-center gap-2 py-24 text-sm text-muted"><UiSpinner /> Memuat undangan…</div>
    <div v-else-if="error" class="card p-8 text-center">
      <p class="text-sm text-danger">{{ errorMessage(error) }}</p>
      <UiButton variant="secondary" size="sm" class="mt-3" @click="load"><UiIcon name="refresh" :size="14" /> Coba lagi</UiButton>
    </div>
    <div v-else-if="!invitations.length" class="card">
      <UiEmpty title="Belum ada undangan" description="Undangan yang Anda buat akan tampil di sini." icon="mail">
        <UiButton v-if="!createBlockReason" @click="createOpen = true"><UiIcon name="plus" :size="16" /> Buat Undangan</UiButton>
      </UiEmpty>
    </div>
    <div v-else class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <InvitationCard v-for="inv in invitations" :key="inv.id" :invitation="inv" />
    </div>

    <CreateInvitationDialog v-model:open="createOpen" />
  </div>
</template>
