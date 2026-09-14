<script setup lang="ts">
import { formatDate, type InvitationStatus, type InvitationSummary } from '@undangan/shared'
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import CreateInvitationDialog from '../components/CreateInvitationDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import UiBadge from '../components/UiBadge.vue'
import UiPagination from '../components/UiPagination.vue'
import UiState from '../components/UiState.vue'
import { INVITATION_STATUS } from '../lib/labels'
import { usePaged } from '../lib/usePaged'
import { formatNumber, useDebounced } from '../lib/util'

const route = useRoute()
const router = useRouter()

const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const qDebounced = useDebounced(q, 350)
const status = ref<InvitationStatus | ''>(
  typeof route.query.status === 'string' && route.query.status in INVITATION_STATUS ? (route.query.status as InvitationStatus) : '',
)
const userId = ref(typeof route.query.user_id === 'string' ? route.query.user_id : '')
const createOpen = ref(false)

const { items, total, page, perPage, loading, error, load } = usePaged<InvitationSummary>('/invitations', () => ({
  q: qDebounced.value.trim(),
  status: status.value,
  user_id: userId.value,
}))

watch([qDebounced, status, userId], () => {
  const query: Record<string, string> = {}
  if (qDebounced.value.trim()) query.q = qDebounced.value.trim()
  if (status.value) query.status = status.value
  if (userId.value) query.user_id = userId.value
  router.replace({ query })
})

function hostOf(url: string | null) {
  if (!url) return ''
  try {
    return new URL(url).host
  } catch {
    return url
  }
}
</script>

<template>
  <div>
    <PageHeader title="Undangan" subtitle="Semua undangan dari seluruh customer.">
      <template #actions>
        <button type="button" class="btn btn-primary" @click="createOpen = true"><AppIcon name="plus" :size="14" /> Buat Undangan</button>
      </template>
    </PageHeader>

    <div class="card">
      <div class="flex flex-wrap items-center gap-2 border-b border-line px-3 py-2.5">
        <div class="relative w-full sm:w-80">
          <AppIcon name="search" class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted" />
          <input v-model="q" type="search" class="input h-8 pl-8" placeholder="Cari judul, subdomain, pemilik…" aria-label="Cari undangan" />
        </div>
        <select v-model="status" class="input h-8 w-auto" aria-label="Filter status">
          <option value="">Semua status</option>
          <option v-for="(s, key) in INVITATION_STATUS" :key="key" :value="key">{{ s.label }}</option>
        </select>
        <span v-if="userId" class="inline-flex h-8 items-center gap-1.5 rounded-md border border-accent/30 bg-accent/10 pr-1 pl-2.5 text-xs text-accent">
          Filter pemilik
          <button type="button" class="grid size-6 place-items-center rounded hover:bg-accent/20" aria-label="Hapus filter pemilik" @click="userId = ''">
            <AppIcon name="x" :size="12" />
          </button>
        </span>
      </div>

      <UiState
        :loading="loading"
        :error="error"
        :empty="!items.length"
        :empty-title="q || status || userId ? 'Tidak ada undangan yang cocok' : 'Belum ada undangan'"
        empty-icon="mail"
        @retry="load"
      >
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>Judul</th>
                <th>Pemilik</th>
                <th>Tema</th>
                <th>Subdomain</th>
                <th>Status</th>
                <th>Tanggal acara</th>
                <th class="text-right">Tamu</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="inv in items" :key="inv.id" class="row-link" @click="router.push(`/invitations/${inv.id}`)">
                <td class="max-w-64">
                  <RouterLink :to="`/invitations/${inv.id}`" class="block truncate font-medium hover:text-accent" @click.stop>{{ inv.title }}</RouterLink>
                  <p class="num text-[11px] text-muted">Diubah {{ formatDate(inv.updated_at) }}</p>
                </td>
                <td class="max-w-52">
                  <RouterLink :to="`/users/${inv.user_id}`" class="block truncate hover:text-accent" @click.stop>{{ inv.user_name }}</RouterLink>
                  <p class="truncate text-xs text-muted">{{ inv.user_email }}</p>
                </td>
                <td>
                  <span v-if="inv.theme" class="inline-flex items-center gap-1 text-muted">
                    {{ inv.theme }}
                    <AppIcon v-if="inv.theme_locked" name="key" :size="11" title="Tema terkunci untuk customer" />
                  </span>
                  <span v-else class="text-muted/60">belum dipilih</span>
                </td>
                <td>
                  <a v-if="inv.url" :href="inv.url" target="_blank" rel="noopener" class="inline-flex items-center gap-1 font-mono text-xs text-accent hover:underline" @click.stop>
                    {{ hostOf(inv.url) }} <AppIcon name="external" :size="11" />
                  </a>
                  <span v-else-if="inv.subdomain" class="font-mono text-xs text-muted">{{ inv.subdomain }}</span>
                  <span v-else class="text-muted/60">-</span>
                </td>
                <td><UiBadge :tone="INVITATION_STATUS[inv.status]?.tone">{{ INVITATION_STATUS[inv.status]?.label ?? inv.status }}</UiBadge></td>
                <td class="num text-muted">{{ formatDate(inv.event_date) }}</td>
                <td class="num text-right">{{ formatNumber(inv.guest_count) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <UiPagination v-model:page="page" :total="total" :per-page="perPage" />
      </UiState>
    </div>

    <CreateInvitationDialog :open="createOpen" @close="createOpen = false" />
  </div>
</template>
