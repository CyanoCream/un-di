<script setup lang="ts">
import { api, type EventType, type Invitation, type Paginated, type ThemeInfo, type User } from '@undangan/shared'
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { EVENT_TYPE_LABEL } from '../lib/labels'
import { toast } from '../lib/toast'
import { errMsg, fieldErrors, initials, itemsOf, useDebounced } from '../lib/util'
import AppIcon from './AppIcon.vue'
import UiDialog from './UiDialog.vue'
import UiSpinner from './UiSpinner.vue'

type PickedUser = Pick<User, 'id' | 'name' | 'email'>

const props = defineProps<{ open: boolean; presetUser?: PickedUser | null }>()
const emit = defineEmits<{ close: [] }>()
const router = useRouter()

const user = ref<PickedUser | null>(null)
const form = reactive<{ event_type: EventType; theme: string; subdomain: string }>({ event_type: 'pernikahan', theme: '', subdomain: '' })
const errors = ref<Record<string, string>>({})
const formError = ref('')
const saving = ref(false)

// Pencarian user
const userQ = ref('')
const userQDebounced = useDebounced(userQ, 300)
const userResults = ref<User[]>([])
const userSearching = ref(false)
const userSearchError = ref('')
let userSeq = 0

watch(userQDebounced, async (q) => {
  const my = ++userSeq
  if (!q.trim()) {
    userResults.value = []
    return
  }
  userSearching.value = true
  userSearchError.value = ''
  try {
    const res = await api.get<Paginated<User>>('/admin/users', { q: q.trim(), role: 'customer', per_page: 8 })
    if (my === userSeq) userResults.value = res.items ?? []
  } catch (e) {
    if (my === userSeq) userSearchError.value = errMsg(e)
  } finally {
    if (my === userSeq) userSearching.value = false
  }
})

// Tema
const themes = ref<ThemeInfo[]>([])
const themesLoading = ref(false)

// Subdomain
type SubCheck = { state: 'idle' | 'checking' | 'ok' | 'taken' | 'error'; reason?: string }
const subCheck = ref<SubCheck>({ state: 'idle' })
const subDebounced = useDebounced(computed(() => form.subdomain), 400)
let subSeq = 0

watch(subDebounced, async (name) => {
  const my = ++subSeq
  const n = name.trim().toLowerCase()
  if (!n) {
    subCheck.value = { state: 'idle' }
    return
  }
  if (!/^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])?$/.test(n) || n.length < 3) {
    subCheck.value = { state: 'taken', reason: 'Gunakan 3–63 huruf kecil, angka, atau tanda hubung (tidak di awal/akhir).' }
    return
  }
  subCheck.value = { state: 'checking' }
  try {
    const res = await api.get<{ name: string; available: boolean; reason?: string }>('/subdomains/check', { name: n })
    if (my !== subSeq) return
    subCheck.value = res.available ? { state: 'ok' } : { state: 'taken', reason: res.reason || 'Subdomain sudah dipakai.' }
  } catch (e) {
    if (my === subSeq) subCheck.value = { state: 'error', reason: errMsg(e) }
  }
})

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    user.value = props.presetUser ?? null
    Object.assign(form, { event_type: 'pernikahan', theme: '', subdomain: '' })
    userQ.value = ''
    userResults.value = []
    errors.value = {}
    formError.value = ''
    subCheck.value = { state: 'idle' }
    if (!themes.value.length) {
      themesLoading.value = true
      try {
        themes.value = itemsOf(await api.get<ThemeInfo[] | { items: ThemeInfo[] }>('/themes'))
      } catch {
        /* tema bisa dipilih nanti */
      } finally {
        themesLoading.value = false
      }
    }
  },
)

async function submit() {
  errors.value = {}
  formError.value = ''
  if (!user.value) {
    errors.value.user_id = 'Pilih pemilik undangan.'
    return
  }
  if (form.subdomain && subCheck.value.state === 'taken') {
    errors.value.subdomain = subCheck.value.reason ?? 'Subdomain tidak tersedia.'
    return
  }
  saving.value = true
  try {
    const inv = await api.post<Invitation>('/invitations', {
      user_id: user.value.id,
      event_type: form.event_type,
      ...(form.theme ? { theme: form.theme } : {}),
      ...(form.subdomain.trim() ? { subdomain: form.subdomain.trim().toLowerCase() } : {}),
    })
    toast.success('Undangan dibuat.')
    emit('close')
    router.push(`/invitations/${inv.id}`)
  } catch (e) {
    errors.value = fieldErrors(e)
    formError.value = Object.keys(errors.value).length ? '' : errMsg(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UiDialog :open="open" title="Buat Undangan" description="Undangan dibuat atas nama customer; data bisa diisi setelahnya." :persistent="saving" @close="emit('close')">
    <form id="create-inv-form" class="space-y-4" novalidate @submit.prevent="submit">
      <p v-if="formError" class="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-xs text-danger" role="alert">{{ formError }}</p>

      <!-- Pemilik -->
      <div>
        <span class="label">Pemilik (customer)</span>
        <div v-if="user" class="flex items-center gap-2.5 rounded-md border border-line bg-canvas px-3 py-2">
          <span class="grid size-7 place-items-center rounded-full bg-accent/15 text-[10px] font-semibold text-accent">{{ initials(user.name) }}</span>
          <div class="min-w-0 flex-1">
            <p class="truncate text-[13px] font-medium">{{ user.name }}</p>
            <p class="truncate text-xs text-muted">{{ user.email }}</p>
          </div>
          <button v-if="!presetUser" type="button" class="btn btn-ghost btn-sm" @click="user = null">Ganti</button>
        </div>
        <div v-else>
          <div class="relative">
            <AppIcon name="search" class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-muted" />
            <input
              v-model="userQ"
              type="search"
              class="input pl-8"
              :class="errors.user_id && 'input-error'"
              placeholder="Cari nama atau email customer…"
              aria-label="Cari customer"
              data-autofocus
            />
            <UiSpinner v-if="userSearching" :size="14" class="absolute top-1/2 right-3 -translate-y-1/2 text-muted" />
          </div>
          <p v-if="errors.user_id" class="field-error">{{ errors.user_id }}</p>
          <p v-if="userSearchError" class="field-error">{{ userSearchError }}</p>
          <ul v-if="userResults.length" class="mt-1.5 max-h-52 overflow-y-auto rounded-md border border-line bg-canvas py-1" role="listbox" aria-label="Hasil pencarian customer">
            <li v-for="u in userResults" :key="u.id">
              <button
                type="button"
                role="option"
                :aria-selected="false"
                class="flex w-full items-center gap-2.5 px-3 py-1.5 text-left hover:bg-surface-2 focus:bg-surface-2 focus:outline-none"
                @click="user = u"
              >
                <span class="grid size-6 place-items-center rounded-full bg-surface-2 text-[9px] font-semibold text-muted">{{ initials(u.name) }}</span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-[13px]">{{ u.name }}</span>
                  <span class="block truncate text-xs text-muted">{{ u.email }}</span>
                </span>
                <span v-if="u.is_suspended" class="text-[11px] text-danger">ditangguhkan</span>
              </button>
            </li>
          </ul>
          <p v-else-if="userQDebounced.trim() && !userSearching && !userSearchError" class="hint">Tidak ada customer yang cocok.</p>
        </div>
      </div>

      <!-- Jenis acara -->
      <fieldset>
        <legend class="label">Jenis acara</legend>
        <div class="grid grid-cols-2 gap-2">
          <label
            v-for="(label, value) in EVENT_TYPE_LABEL"
            :key="value"
            class="flex cursor-pointer items-center gap-2 rounded-md border px-3 py-2 text-[13px] transition-colors"
            :class="form.event_type === value ? 'border-accent bg-accent/10 text-ink' : 'border-line text-muted hover:text-ink'"
          >
            <input v-model="form.event_type" type="radio" name="event_type" :value="value" class="accent-[var(--color-accent)]" />
            {{ label }}
          </label>
        </div>
      </fieldset>

      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label for="ci-theme" class="label">Tema (opsional)</label>
          <select id="ci-theme" v-model="form.theme" class="input" :disabled="themesLoading">
            <option value="">— Pilih nanti —</option>
            <option v-for="t in themes" :key="t.slug" :value="t.slug">{{ t.name }}{{ t.is_premium ? ' ★' : '' }}</option>
          </select>
          <p v-if="errors.theme" class="field-error">{{ errors.theme }}</p>
        </div>
        <div>
          <label for="ci-sub" class="label">Subdomain (opsional)</label>
          <input
            id="ci-sub"
            v-model="form.subdomain"
            class="input font-mono"
            :class="(errors.subdomain || subCheck.state === 'taken') && 'input-error'"
            placeholder="raka-nadia"
            autocomplete="off"
            spellcheck="false"
          />
          <p v-if="errors.subdomain" class="field-error">{{ errors.subdomain }}</p>
          <p v-else-if="subCheck.state === 'checking'" class="hint inline-flex items-center gap-1"><UiSpinner :size="11" /> Memeriksa…</p>
          <p v-else-if="subCheck.state === 'ok'" class="mt-1 inline-flex items-center gap-1 text-xs text-success"><AppIcon name="check" :size="12" /> Tersedia</p>
          <p v-else-if="subCheck.state === 'taken' || subCheck.state === 'error'" class="field-error">{{ subCheck.reason }}</p>
        </div>
      </div>
      <p v-if="form.theme" class="flex items-start gap-1.5 text-xs text-muted">
        <AppIcon name="info" :size="13" class="mt-px" /> Tema yang dipilih admin langsung terkunci bagi customer.
      </p>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">Batal</button>
      <button type="submit" form="create-inv-form" class="btn btn-primary" :disabled="saving || subCheck.state === 'checking'">
        <UiSpinner v-if="saving" :size="14" /> Buat undangan
      </button>
    </template>
  </UiDialog>
</template>
