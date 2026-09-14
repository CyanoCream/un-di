<script setup lang="ts">
import { formatDate, type Role, type User } from '@undangan/shared'
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import CreateUserDialog from '../components/CreateUserDialog.vue'
import PageHeader from '../components/PageHeader.vue'
import UiBadge from '../components/UiBadge.vue'
import UiPagination from '../components/UiPagination.vue'
import UiState from '../components/UiState.vue'
import { ROLE_LABEL } from '../lib/labels'
import { usePaged } from '../lib/usePaged'
import { initials, useDebounced } from '../lib/util'

const route = useRoute()
const router = useRouter()

const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const role = ref<Role | ''>(route.query.role === 'customer' || route.query.role === 'super_admin' ? route.query.role : '')
const qDebounced = useDebounced(q, 350)
const createOpen = ref(false)

const { items, total, page, perPage, loading, error, load } = usePaged<User>('/admin/users', () => ({ q: qDebounced.value.trim(), role: role.value }))

// Pencarian global di top bar mengubah ?q= tanpa me-mount ulang halaman.
watch(
  () => route.query.q,
  (v) => {
    const next = typeof v === 'string' ? v : ''
    if (next !== qDebounced.value) {
      q.value = next
      qDebounced.value = next
    }
  },
)
watch([qDebounced, role], () => {
  const query: Record<string, string> = {}
  if (qDebounced.value.trim()) query.q = qDebounced.value.trim()
  if (role.value) query.role = role.value
  router.replace({ query })
})

function onCreated(u: User) {
  createOpen.value = false
  router.push(`/users/${u.id}`)
}
</script>

<template>
  <div>
    <PageHeader title="Pengguna" subtitle="Customer dan super admin terdaftar.">
      <template #actions>
        <button type="button" class="btn btn-primary" @click="createOpen = true"><AppIcon name="plus" :size="14" /> Tambah Customer</button>
      </template>
    </PageHeader>

    <div class="card">
      <div class="flex flex-wrap items-center gap-2 border-b border-line px-3 py-2.5">
        <div class="relative w-full sm:w-80">
          <AppIcon name="search" class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted" />
          <input v-model="q" type="search" class="input h-8 pl-8" placeholder="Cari nama, email, no. HP…" aria-label="Cari pengguna" />
        </div>
        <select v-model="role" class="input h-8 w-auto" aria-label="Filter role">
          <option value="">Semua role</option>
          <option value="customer">Customer</option>
          <option value="super_admin">Super admin</option>
        </select>
      </div>

      <UiState
        :loading="loading"
        :error="error"
        :empty="!items.length"
        :empty-title="q || role ? 'Tidak ada pengguna yang cocok' : 'Belum ada pengguna'"
        empty-icon="users"
        @retry="load"
      >
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>Nama</th>
                <th>No. HP</th>
                <th>Role</th>
                <th>Status</th>
                <th>Terdaftar</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="u in items" :key="u.id" class="row-link" @click="router.push(`/users/${u.id}`)">
                <td class="max-w-80">
                  <RouterLink :to="`/users/${u.id}`" class="flex items-center gap-2.5" @click.stop>
                    <span class="grid size-7 shrink-0 place-items-center rounded-full bg-surface-2 text-[10px] font-semibold text-muted ring-1 ring-line">
                      {{ initials(u.name) }}
                    </span>
                    <span class="min-w-0">
                      <span class="block truncate font-medium hover:text-accent">{{ u.name }}</span>
                      <span class="block truncate text-xs text-muted">{{ u.email }}</span>
                    </span>
                  </RouterLink>
                </td>
                <td class="num text-muted">{{ u.phone || '-' }}</td>
                <td>
                  <UiBadge :tone="u.role === 'super_admin' ? 'accent' : 'neutral'" :dot="false">{{ ROLE_LABEL[u.role] ?? u.role }}</UiBadge>
                </td>
                <td>
                  <UiBadge v-if="u.is_suspended" tone="danger">Ditangguhkan</UiBadge>
                  <UiBadge v-else tone="success">Aktif</UiBadge>
                </td>
                <td class="num text-muted">{{ formatDate(u.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <UiPagination v-model:page="page" :total="total" :per-page="perPage" />
      </UiState>
    </div>

    <CreateUserDialog :open="createOpen" @close="createOpen = false" @created="onCreated" />
  </div>
</template>
