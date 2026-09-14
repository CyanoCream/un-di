<script setup lang="ts">
import { formatDate, type AuditLog } from '@undangan/shared'
import { ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import UiPagination from '../components/UiPagination.vue'
import UiState from '../components/UiState.vue'
import { usePaged } from '../lib/usePaged'
import { relativeTime, useDebounced } from '../lib/util'

const q = ref('')
const qDebounced = useDebounced(q, 350)
const { items, total, page, perPage, loading, error, load } = usePaged<AuditLog>('/admin/audit-logs', () => ({ q: qDebounced.value.trim() }), 50)

const expanded = ref(new Set<number>())
function toggle(id: number) {
  const s = new Set(expanded.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  expanded.value = s
}

function actionTone(action: string) {
  if (/(delete|reject|suspend|remove|unpublish)/.test(action)) return 'text-danger'
  if (/(approve|create|grant|publish|activate)/.test(action)) return 'text-success'
  if (/(update|reset|theme|subdomain|patch)/.test(action)) return 'text-warning'
  return 'text-accent'
}

function targetLink(target: string): string | null {
  const [kind, id] = target.split(':')
  if (!id) return null
  if (kind === 'user') return `/users/${id}`
  if (kind === 'invitation') return `/invitations/${id}`
  if (kind === 'order') return `/orders?open=${id}`
  return null
}

function metaKeys(meta: Record<string, unknown> | null) {
  return meta ? Object.keys(meta).length : 0
}
</script>

<template>
  <div>
    <PageHeader title="Audit Log" subtitle="Jejak aksi admin yang mengubah data customer." />

    <div class="card">
      <div class="flex flex-wrap items-center gap-2 border-b border-line px-3 py-2.5">
        <div class="relative w-full sm:w-80">
          <AppIcon name="search" class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted" />
          <input v-model="q" type="search" class="input h-8 pl-8" placeholder="Cari aksi, aktor, target…" aria-label="Cari audit log" />
        </div>
        <button type="button" class="btn btn-ghost btn-sm ml-auto" @click="load"><AppIcon name="refresh" :size="13" /> Muat ulang</button>
      </div>

      <UiState :loading="loading" :error="error" :empty="!items.length" empty-title="Belum ada catatan" empty-icon="audit" @retry="load">
        <div class="table-wrap">
          <table class="data-table font-mono text-xs">
            <thead>
              <tr>
                <th class="font-sans">Waktu</th>
                <th class="font-sans">Aktor</th>
                <th class="font-sans">Aksi</th>
                <th class="font-sans">Target</th>
                <th class="font-sans">Meta</th>
              </tr>
            </thead>
            <tbody>
              <template v-for="log in items" :key="log.id">
                <tr>
                  <td class="num" :title="formatDate(log.created_at, true)">
                    <span class="text-ink">{{ formatDate(log.created_at, true) }}</span>
                    <span class="ml-2 font-sans text-[11px] text-muted">{{ relativeTime(log.created_at) }}</span>
                  </td>
                  <td class="font-sans text-[13px]">{{ log.actor_name ?? 'sistem' }}</td>
                  <td :class="actionTone(log.action)">{{ log.action }}</td>
                  <td class="max-w-72 truncate text-muted">
                    <RouterLink v-if="targetLink(log.target)" :to="targetLink(log.target)!" class="hover:text-accent hover:underline">{{ log.target }}</RouterLink>
                    <span v-else>{{ log.target }}</span>
                  </td>
                  <td>
                    <button
                      v-if="metaKeys(log.meta)"
                      type="button"
                      class="inline-flex items-center gap-1 rounded px-1.5 py-0.5 font-sans text-[11px] text-muted ring-1 ring-line hover:text-ink"
                      :aria-expanded="expanded.has(log.id)"
                      @click="toggle(log.id)"
                    >
                      <AppIcon :name="expanded.has(log.id) ? 'chevron-down' : 'chevron-right'" :size="12" />
                      {{ metaKeys(log.meta) }} field
                    </button>
                    <span v-else class="text-muted/50">—</span>
                  </td>
                </tr>
                <tr v-if="expanded.has(log.id)">
                  <td colspan="5" class="bg-canvas/60 !whitespace-normal">
                    <pre class="max-h-72 overflow-auto rounded-md p-2 text-[11px] leading-relaxed text-ink/90">{{ JSON.stringify(log.meta, null, 2) }}</pre>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
        <UiPagination v-model:page="page" :total="total" :per-page="perPage" />
      </UiState>
    </div>
  </div>
</template>
