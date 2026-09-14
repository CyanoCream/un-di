<script setup lang="ts">
import { api, formatRupiah, type Plan } from '@undangan/shared'
import { computed, onMounted, reactive, ref } from 'vue'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import UiBadge from '../components/UiBadge.vue'
import UiDialog from '../components/UiDialog.vue'
import UiSpinner from '../components/UiSpinner.vue'
import UiState from '../components/UiState.vue'
import UiToggle from '../components/UiToggle.vue'
import { toast } from '../lib/toast'
import { errMsg, fieldErrors, formatNumber, itemsOf } from '../lib/util'

const plans = ref<Plan[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    plans.value = itemsOf(await api.get<{ items: Plan[] }>('/admin/plans'))
  } catch (e) {
    error.value = errMsg(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)

const sorted = computed(() => [...plans.value].sort((a, b) => a.sort_order - b.sort_order || a.price - b.price))

type PlanForm = Omit<Plan, 'id'>
const blank = (): PlanForm => ({
  name: '',
  description: '',
  price: 0,
  duration_days: 365,
  grace_days: 7,
  max_invitations: 1,
  max_guests: 500,
  allow_custom_domain: false,
  allow_checkin: false,
  is_active: true,
  sort_order: (plans.value.at(-1)?.sort_order ?? 0) + 10,
})

const dialogOpen = ref(false)
const editing = ref<Plan | null>(null)
const form = reactive<PlanForm>(blank())
const errors = ref<Record<string, string>>({})
const formError = ref('')
const saving = ref(false)

function openCreate() {
  editing.value = null
  Object.assign(form, blank())
  errors.value = {}
  formError.value = ''
  dialogOpen.value = true
}
function openEdit(p: Plan) {
  editing.value = p
  const { id: _id, ...rest } = p
  Object.assign(form, rest)
  errors.value = {}
  formError.value = ''
  dialogOpen.value = true
}

function validate() {
  const e: Record<string, string> = {}
  if (!form.name.trim()) e.name = 'Nama paket wajib diisi.'
  if (!Number.isFinite(form.price) || form.price < 0) e.price = 'Harga tidak valid.'
  if (!Number.isInteger(form.duration_days) || form.duration_days < 1) e.duration_days = 'Minimal 1 hari.'
  if (!Number.isInteger(form.grace_days) || form.grace_days < 0) e.grace_days = 'Minimal 0 hari.'
  if (!Number.isInteger(form.max_invitations) || form.max_invitations < 1) e.max_invitations = 'Minimal 1.'
  if (!Number.isInteger(form.max_guests) || form.max_guests < 0) e.max_guests = 'Minimal 0.'
  if (!Number.isInteger(form.sort_order)) e.sort_order = 'Harus bilangan bulat.'
  errors.value = e
  return !Object.keys(e).length
}

async function save() {
  if (!validate()) return
  saving.value = true
  formError.value = ''
  const body = { ...form, name: form.name.trim(), description: form.description.trim() }
  try {
    if (editing.value) {
      const p = await api.patch<Plan>(`/admin/plans/${editing.value.id}`, body)
      plans.value = plans.value.map((x) => (x.id === p.id ? p : x))
      toast.success(`Paket ${p.name} diperbarui.`)
    } else {
      const p = await api.post<Plan>('/admin/plans', body)
      plans.value.push(p)
      toast.success(`Paket ${p.name} dibuat.`)
    }
    dialogOpen.value = false
  } catch (e) {
    errors.value = fieldErrors(e)
    formError.value = Object.keys(errors.value).length ? '' : errMsg(e)
  } finally {
    saving.value = false
  }
}

const NUM_FIELDS = [
  { key: 'duration_days', label: 'Durasi', suffix: 'hari', min: 1 },
  { key: 'grace_days', label: 'Masa tenggang', suffix: 'hari', min: 0 },
  { key: 'max_invitations', label: 'Maks. undangan', suffix: 'undangan', min: 1 },
  { key: 'max_guests', label: 'Maks. tamu', suffix: 'tamu / undangan', min: 0 },
] as const
</script>

<template>
  <div>
    <PageHeader title="Paket" subtitle="Paket langganan yang dijual. Paket tidak dihapus — nonaktifkan agar tidak tampil di checkout.">
      <template #actions>
        <button type="button" class="btn btn-primary" @click="openCreate"><AppIcon name="plus" :size="14" /> Paket baru</button>
      </template>
    </PageHeader>

    <div class="card">
      <UiState :loading="loading" :error="error" :empty="!plans.length" empty-title="Belum ada paket" empty-text="Buat paket pertama agar customer bisa berlangganan." @retry="load">
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th class="w-12 text-right">Urut</th>
                <th>Paket</th>
                <th class="text-right">Harga</th>
                <th class="text-right">Durasi</th>
                <th class="text-right">Tenggang</th>
                <th class="text-right">Maks. undangan</th>
                <th class="text-right">Maks. tamu</th>
                <th>Domain kustom</th>
                <th>Check-in</th>
                <th>Status</th>
                <th class="w-12"><span class="sr-only">Aksi</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in sorted" :key="p.id" class="row-link" :class="!p.is_active && 'opacity-60'" @click="openEdit(p)">
                <td class="num text-right text-muted">{{ p.sort_order }}</td>
                <td class="max-w-72">
                  <p class="font-medium">{{ p.name }}</p>
                  <p class="truncate text-xs text-muted">{{ p.description || '—' }}</p>
                </td>
                <td class="num text-right font-medium">{{ formatRupiah(p.price) }}</td>
                <td class="num text-right">{{ p.duration_days }} hr</td>
                <td class="num text-right text-muted">{{ p.grace_days }} hr</td>
                <td class="num text-right">{{ formatNumber(p.max_invitations) }}</td>
                <td class="num text-right">{{ formatNumber(p.max_guests) }}</td>
                <td>
                  <AppIcon v-if="p.allow_custom_domain" name="check" class="text-success" />
                  <span v-else class="text-muted/60">—</span>
                </td>
                <td>
                  <AppIcon v-if="p.allow_checkin" name="check" class="text-success" />
                  <span v-else class="text-muted/60">—</span>
                </td>
                <td>
                  <UiBadge :tone="p.is_active ? 'success' : 'neutral'">{{ p.is_active ? 'Aktif' : 'Nonaktif' }}</UiBadge>
                </td>
                <td>
                  <button type="button" class="btn btn-ghost btn-sm btn-icon" :aria-label="`Ubah paket ${p.name}`" @click.stop="openEdit(p)">
                    <AppIcon name="edit" :size="14" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </UiState>
    </div>

    <UiDialog :open="dialogOpen" :title="editing ? `Ubah paket ${editing.name}` : 'Paket baru'" size="lg" :persistent="saving" @close="dialogOpen = false">
      <form id="plan-form" class="space-y-4" novalidate @submit.prevent="save">
        <p v-if="formError" class="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-xs text-danger" role="alert">{{ formError }}</p>
        <div class="grid gap-4 sm:grid-cols-3">
          <div class="sm:col-span-2">
            <label for="pl-name" class="label">Nama paket</label>
            <input id="pl-name" v-model="form.name" class="input" :class="errors.name && 'input-error'" placeholder="Premium" data-autofocus />
            <p v-if="errors.name" class="field-error">{{ errors.name }}</p>
          </div>
          <div>
            <label for="pl-price" class="label">Harga (Rp)</label>
            <input id="pl-price" v-model.number="form.price" type="number" min="0" step="1000" class="input num" :class="errors.price && 'input-error'" />
            <p v-if="errors.price" class="field-error">{{ errors.price }}</p>
            <p v-else class="hint num">{{ formatRupiah(form.price || 0) }}</p>
          </div>
        </div>
        <div>
          <label for="pl-desc" class="label">Deskripsi</label>
          <textarea id="pl-desc" v-model="form.description" rows="2" class="input" placeholder="Ringkasan fitur yang tampil di halaman paket" />
        </div>
        <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div v-for="f in NUM_FIELDS" :key="f.key">
            <label :for="`pl-${f.key}`" class="label">{{ f.label }}</label>
            <input :id="`pl-${f.key}`" v-model.number="form[f.key]" type="number" :min="f.min" class="input num" :class="errors[f.key] && 'input-error'" />
            <p v-if="errors[f.key]" class="field-error">{{ errors[f.key] }}</p>
            <p v-else class="hint">{{ f.suffix }}</p>
          </div>
        </div>
        <div class="grid gap-4 border-t border-line pt-4 sm:grid-cols-2">
          <UiToggle v-model="form.is_active" label="Aktif (tampil di checkout)" tone="success" />
          <UiToggle v-model="form.allow_custom_domain" label="Izinkan domain kustom" />
          <UiToggle v-model="form.allow_checkin" label="Check-in QR di venue" />
          <div class="flex items-center gap-2">
            <label for="pl-sort" class="text-sm whitespace-nowrap">Urutan</label>
            <input id="pl-sort" v-model.number="form.sort_order" type="number" class="input num h-8 w-24" :class="errors.sort_order && 'input-error'" />
          </div>
        </div>
        <p v-if="editing" class="flex items-start gap-1.5 text-xs text-muted">
          <AppIcon name="info" :size="13" class="mt-px" /> Perubahan kuota/durasi berlaku untuk langganan baru atau perpanjangan berikutnya.
        </p>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="dialogOpen = false">Batal</button>
        <button type="submit" form="plan-form" class="btn btn-primary" :disabled="saving">
          <UiSpinner v-if="saving" :size="14" /> {{ editing ? 'Simpan perubahan' : 'Buat paket' }}
        </button>
      </template>
    </UiDialog>
  </div>
</template>
