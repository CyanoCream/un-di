<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, useId, watch } from 'vue'
import { ApiError, api, formatDate } from '../api'
import { confirm } from '../composables/useConfirm'
import { toast } from '../composables/useToast'
import { fillShareTemplate, waLink } from '../content'
import type { Guest, GuestInput, ImportResult, Invitation, Paginated } from '../types'
import { copyText, defaultWhatsappTemplate, errorMessage } from '../utils/misc'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiDialog from './ui/UiDialog.vue'
import UiEmpty from './ui/UiEmpty.vue'
import UiField from './ui/UiField.vue'
import UiIcon from './ui/UiIcon.vue'
import UiInput from './ui/UiInput.vue'
import UiPagination from './ui/UiPagination.vue'
import UiSpinner from './ui/UiSpinner.vue'

const props = defineProps<{ invitation: Invitation }>()
const emit = defineEmits<{ changed: [] }>()

const base = computed(() => `/invitations/${props.invitation.id}/guests`)
const PER_PAGE = 25

// ---------- Daftar ----------
const q = ref('')
const group = ref('')
const page = ref(1)
const data = ref<Paginated<Guest>>({ items: [], total: 0, page: 1, per_page: PER_PAGE })
const loading = ref(false)
const loaded = ref(false)
const knownGroups = reactive(new Set<string>())
let reqSeq = 0

async function load() {
  const my = ++reqSeq
  loading.value = true
  try {
    const res = await api.get<Paginated<Guest>>(base.value, { q: q.value.trim(), group: group.value, page: page.value, per_page: PER_PAGE })
    if (my !== reqSeq) return
    data.value = { ...res, items: res.items ?? [] }
    res.items?.forEach((g) => g.group_name && knownGroups.add(g.group_name))
    loaded.value = true
  } catch (e) {
    if (my === reqSeq) toast.error(e)
  } finally {
    if (my === reqSeq) loading.value = false
  }
}

let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(q, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    load()
  }, 300)
})
watch(group, () => {
  page.value = 1
  load()
})
watch(page, load)
watch(() => props.invitation.id, () => {
  knownGroups.clear()
  page.value = 1
  load()
})
onMounted(load)
onBeforeUnmount(() => searchTimer && clearTimeout(searchTimer))

const groups = computed(() => [...knownGroups].sort((a, b) => a.localeCompare(b)))
const filtering = computed(() => !!q.value.trim() || !!group.value)

// ---------- Kuota ----------
const quota = computed(() => props.invitation.quota ?? { max_guests: 0, used_guests: 0 })
const quotaPct = computed(() => (quota.value.max_guests > 0 ? Math.min(100, (quota.value.used_guests / quota.value.max_guests) * 100) : 0))
const quotaFull = computed(() => quota.value.max_guests > 0 && quota.value.used_guests >= quota.value.max_guests)

// ---------- Aksi per tamu ----------
function shareText(g: Guest) {
  const tpl = props.invitation.content?.share?.whatsapp_template || defaultWhatsappTemplate(props.invitation.content)
  return fillShareTemplate(tpl, g.name, g.link ?? '')
}

/** Tampilan ringkas link personal (…/nama-tamu) tanpa skema. */
function linkPath(g: Guest) {
  return (g.link ?? '').replace(/^https?:\/\//, '')
}

async function copyLink(g: Guest) {
  if (!g.link) {
    toast.info('Atur subdomain undangan terlebih dahulu agar link tamu tersedia.')
    return
  }
  if (await copyText(g.link)) toast.success(`Link untuk ${g.name} disalin`)
  else toast.error('Gagal menyalin link')
}

async function shareWa(g: Guest) {
  if (!g.link) {
    toast.info('Atur subdomain undangan terlebih dahulu agar link tamu tersedia.')
    return
  }
  const text = shareText(g)
  if (g.phone) {
    window.open(waLink(g.phone, text), '_blank', 'noopener')
  } else if (await copyText(text)) {
    toast.success(`${g.name} belum punya no. HP. Pesan undangan disalin — tempel di WhatsApp.`)
  } else {
    toast.error('Gagal menyalin pesan')
  }
}

async function remove(g: Guest) {
  const ok = await confirm({ title: 'Hapus tamu?', message: `${g.name} akan dihapus dari daftar tamu. Link personalnya tidak berlaku lagi.`, danger: true })
  if (!ok) return
  try {
    await api.del(`${base.value}/${g.id}`)
    toast.success('Tamu dihapus')
    if (data.value.items.length === 1 && page.value > 1) page.value--
    else load()
    emit('changed')
  } catch (e) {
    toast.error(e)
  }
}

// ---------- Tambah / edit ----------
const formOpen = ref(false)
const editing = ref<Guest | null>(null)
const form = reactive<GuestInput>({ name: '', phone: '', group_name: '', pax: 1 })
const formErrors = ref<Record<string, string>>({})
const saving = ref(false)
const groupListId = `guest-groups-${useId()}`

function openForm(g: Guest | null = null) {
  editing.value = g
  Object.assign(form, g ? { name: g.name, phone: g.phone, group_name: g.group_name, pax: g.pax || 1 } : { name: '', phone: '', group_name: form.group_name, pax: 1 })
  formErrors.value = {}
  formOpen.value = true
}

async function submitForm(again = false) {
  formErrors.value = {}
  if (!form.name.trim()) {
    formErrors.value = { name: 'Nama wajib diisi' }
    return
  }
  saving.value = true
  const body: GuestInput = { name: form.name.trim(), phone: form.phone.trim(), group_name: form.group_name.trim(), pax: Math.max(1, Number(form.pax) || 1) }
  try {
    if (editing.value) {
      await api.patch<Guest>(`${base.value}/${editing.value.id}`, body)
      toast.success('Data tamu diperbarui')
    } else {
      await api.post<Guest>(base.value, body)
      toast.success(`${body.name} ditambahkan`)
    }
    if (body.group_name) knownGroups.add(body.group_name)
    emit('changed')
    load()
    if (again && !editing.value) {
      Object.assign(form, { name: '', phone: '', pax: 1 })
    } else {
      formOpen.value = false
    }
  } catch (e) {
    if (e instanceof ApiError && Object.keys(e.fields).length) formErrors.value = e.fields
    else toast.error(e)
  } finally {
    saving.value = false
  }
}

// ---------- Import ----------
const importOpen = ref(false)
const importStep = ref<1 | 2>(1)
const importFile = ref<File | null>(null)
const importPreview = ref<ImportResult | null>(null)
const importing = ref(false)
const importError = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const templateUrl = computed(() => api.url(`${base.value}/template.csv`))
const templateXlsxUrl = computed(() => api.url(`${base.value}/template.xlsx`))
const exportUrl = computed(() => api.url(`${base.value}/export.xlsx`))
const MAX_IMPORT_MB = 2

function openImport() {
  importStep.value = 1
  importFile.value = null
  importPreview.value = null
  importError.value = ''
  importOpen.value = true
}

async function onImportFile(e: Event) {
  const el = e.target as HTMLInputElement
  const file = el.files?.[0]
  el.value = ''
  if (!file) return
  importError.value = ''
  if (!/\.(csv|xlsx)$/i.test(file.name)) {
    importError.value = 'Format file harus .csv atau .xlsx'
    return
  }
  if (file.size > MAX_IMPORT_MB * 1024 * 1024) {
    importError.value = `Ukuran file maksimal ${MAX_IMPORT_MB} MB`
    return
  }
  importFile.value = file
  importing.value = true
  try {
    importPreview.value = await api.upload<ImportResult>(`${base.value}/import`, file, {}, { dry_run: 1 })
    importStep.value = 2
  } catch (err) {
    importError.value = errorMessage(err)
  } finally {
    importing.value = false
  }
}

async function doImport() {
  if (!importFile.value || !importPreview.value) return
  importing.value = true
  try {
    const res = await api.upload<ImportResult>(`${base.value}/import`, importFile.value)
    toast.success(`${res.inserted} tamu berhasil diimport`)
    importOpen.value = false
    page.value = 1
    load()
    emit('changed')
  } catch (err) {
    importError.value = errorMessage(err)
    toast.error(err)
  } finally {
    importing.value = false
  }
}

const STATUS = {
  ok: { tone: 'success', label: 'Siap' },
  duplicate: { tone: 'warning', label: 'Duplikat' },
  error: { tone: 'danger', label: 'Error' },
} as const

const quotaLeft = computed(() => (quota.value.max_guests > 0 ? Math.max(0, quota.value.max_guests - quota.value.used_guests) : Infinity))
const importOverQuota = computed(() => !!importPreview.value && importPreview.value.summary.ok > quotaLeft.value)
</script>

<template>
  <div class="space-y-4">
    <!-- Kuota -->
    <div class="rounded-2xl border border-line bg-surface p-4">
      <div class="flex items-center justify-between gap-3 text-sm">
        <span class="flex items-center gap-2 font-medium text-ink"><UiIcon name="users" :size="16" class="text-accent" /> Kuota tamu</span>
        <span class="text-muted">
          <strong class="text-ink">{{ quota.used_guests }}</strong>
          <template v-if="quota.max_guests > 0"> / {{ quota.max_guests }}</template>
          <template v-else> · tanpa batas</template>
        </span>
      </div>
      <div v-if="quota.max_guests > 0" class="mt-2.5 h-2 overflow-hidden rounded-full bg-surface-2">
        <div
          class="h-full rounded-full transition-all"
          :class="quotaPct >= 100 ? 'bg-danger' : quotaPct >= 85 ? 'bg-warning' : 'bg-accent'"
          :style="{ width: `${quotaPct}%` }"
        />
      </div>
      <p v-if="quotaFull" class="mt-2 text-xs text-danger">Kuota tamu penuh. Upgrade paket untuk menambah tamu.</p>
      <p v-if="!invitation.subdomain" class="mt-2 flex items-center gap-1.5 text-xs text-warning">
        <UiIcon name="alert" :size="13" /> Subdomain belum diatur — link personal tamu belum tersedia.
      </p>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="flex flex-1 gap-2">
        <div class="relative min-w-0 flex-1">
          <UiIcon name="search" :size="16" class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
          <UiInput v-model="q" type="search" placeholder="Cari nama atau no. HP…" class="pl-9!" />
        </div>
        <select
          v-model="group"
          class="h-10 max-w-[40%] min-w-0 rounded-xl border border-line bg-surface-2 px-3 text-sm text-ink outline-none focus:border-accent sm:max-w-52"
          aria-label="Filter grup"
        >
          <option value="">Semua grup</option>
          <option v-for="g in groups" :key="g" :value="g">{{ g }}</option>
        </select>
      </div>
      <div class="flex flex-wrap gap-2">
        <UiButton :disabled="quotaFull" class="flex-1 sm:flex-none" @click="openForm()"><UiIcon name="plus" :size="16" /> Tambah Tamu</UiButton>
        <UiButton variant="secondary" :disabled="quotaFull" class="flex-1 sm:flex-none" @click="openImport"><UiIcon name="upload" :size="16" /> Import Excel/CSV</UiButton>
        <UiButton variant="secondary" :href="exportUrl" class="flex-1 sm:flex-none" download><UiIcon name="download" :size="16" /> Export Excel</UiButton>
      </div>
    </div>

    <!-- Daftar -->
    <div class="relative overflow-hidden rounded-2xl border border-line bg-surface">
      <div v-if="loading && loaded" class="absolute inset-x-0 top-0 z-10 h-0.5 animate-pulse bg-accent" />

      <div v-if="!loaded" class="flex items-center justify-center gap-2 py-16 text-sm text-muted"><UiSpinner /> Memuat tamu…</div>

      <UiEmpty
        v-else-if="!data.items.length"
        :title="filtering ? 'Tidak ada tamu yang cocok' : 'Belum ada tamu'"
        :description="filtering ? 'Coba kata kunci atau grup lain.' : 'Tambahkan tamu satu per satu atau import dari Excel/CSV untuk membuat link undangan personal.'"
        icon="users"
      >
        <template v-if="!filtering">
          <UiButton @click="openForm()"><UiIcon name="plus" :size="16" /> Tambah Tamu</UiButton>
          <UiButton variant="secondary" @click="openImport"><UiIcon name="upload" :size="16" /> Import</UiButton>
        </template>
      </UiEmpty>

      <template v-else>
        <!-- Desktop: tabel -->
        <div class="hidden overflow-x-auto md:block">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-line bg-surface-2/60 text-left text-xs font-medium tracking-wide text-muted uppercase">
                <th class="px-4 py-2.5">Nama</th>
                <th class="px-4 py-2.5">Kode</th>
                <th class="px-4 py-2.5">Grup</th>
                <th class="px-4 py-2.5 text-center">Jumlah</th>
                <th class="px-4 py-2.5">No. HP</th>
                <th class="px-4 py-2.5">Status</th>
                <th class="px-4 py-2.5 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-line">
              <tr v-for="g in data.items" :key="g.id" class="transition hover:bg-surface-2/50">
                <td class="px-4 py-2.5">
                  <p class="font-medium text-ink">{{ g.name }}</p>
                  <p v-if="g.link" class="max-w-64 truncate text-[11px] text-muted" :title="g.link">{{ linkPath(g) }}</p>
                </td>
                <td class="px-4 py-2.5">
                  <span class="rounded-md bg-surface-2 px-1.5 py-0.5 font-mono text-xs tracking-wider text-ink" title="Kode undangan (passcode & QR check-in)">{{ g.code }}</span>
                </td>
                <td class="px-4 py-2.5 text-muted">{{ g.group_name || '-' }}</td>
                <td class="px-4 py-2.5 text-center text-ink">{{ g.pax }}</td>
                <td class="px-4 py-2.5 text-muted">{{ g.phone || '-' }}</td>
                <td class="px-4 py-2.5">
                  <UiBadge v-if="g.checked_in_at" tone="accent" :title="formatDate(g.checked_in_at, true)"><UiIcon name="check" :size="12" /> Sudah datang</UiBadge>
                  <UiBadge v-else-if="g.opened_at" tone="success" :title="formatDate(g.opened_at, true)"><UiIcon name="eye" :size="12" /> Dibuka</UiBadge>
                  <UiBadge v-else>Belum dibuka</UiBadge>
                </td>
                <td class="px-4 py-2.5">
                  <div class="flex justify-end gap-0.5">
                    <button type="button" class="rounded-lg p-2 text-muted hover:bg-surface-2 hover:text-ink" title="Salin link" @click="copyLink(g)"><UiIcon name="link" :size="16" /></button>
                    <button type="button" class="rounded-lg p-2 text-muted hover:bg-success/10 hover:text-success" title="Kirim via WhatsApp" @click="shareWa(g)"><UiIcon name="message" :size="16" /></button>
                    <button type="button" class="rounded-lg p-2 text-muted hover:bg-surface-2 hover:text-ink" title="Edit" @click="openForm(g)"><UiIcon name="pencil" :size="16" /></button>
                    <button type="button" class="rounded-lg p-2 text-muted hover:bg-danger/10 hover:text-danger" title="Hapus" @click="remove(g)"><UiIcon name="trash" :size="16" /></button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Mobile: kartu -->
        <ul class="divide-y divide-line md:hidden">
          <li v-for="g in data.items" :key="g.id" class="p-4">
            <div class="flex items-start gap-3">
              <div class="min-w-0 flex-1">
                <p class="truncate font-medium text-ink">{{ g.name }}</p>
                <p class="mt-0.5 text-xs text-muted">
                  {{ g.pax }} orang<template v-if="g.group_name"> · {{ g.group_name }}</template><template v-if="g.phone"> · {{ g.phone }}</template>
                </p>
                <p class="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-muted">
                  <span class="shrink-0">Kode</span>
                  <span class="shrink-0 rounded-md bg-surface-2 px-1.5 py-0.5 font-mono tracking-wider text-ink">{{ g.code }}</span>
                  <span v-if="g.link" class="truncate">{{ linkPath(g) }}</span>
                </p>
              </div>
              <UiBadge v-if="g.checked_in_at" tone="accent">Datang</UiBadge>
              <UiBadge v-else-if="g.opened_at" tone="success">Dibuka</UiBadge>
              <UiBadge v-else>Belum</UiBadge>
            </div>
            <div class="mt-3 grid grid-cols-4 gap-2">
              <button type="button" class="flex flex-col items-center gap-1 rounded-xl bg-surface-2 py-2 text-[11px] text-ink" @click="copyLink(g)"><UiIcon name="link" :size="16" />Salin</button>
              <button type="button" class="flex flex-col items-center gap-1 rounded-xl bg-success/10 py-2 text-[11px] text-success" @click="shareWa(g)"><UiIcon name="message" :size="16" />WhatsApp</button>
              <button type="button" class="flex flex-col items-center gap-1 rounded-xl bg-surface-2 py-2 text-[11px] text-ink" @click="openForm(g)"><UiIcon name="pencil" :size="16" />Edit</button>
              <button type="button" class="flex flex-col items-center gap-1 rounded-xl bg-danger/10 py-2 text-[11px] text-danger" @click="remove(g)"><UiIcon name="trash" :size="16" />Hapus</button>
            </div>
          </li>
        </ul>
      </template>

      <div v-if="data.total > PER_PAGE" class="border-t border-line px-4 py-3">
        <UiPagination v-model:page="page" :total="data.total" :per-page="PER_PAGE" />
      </div>
    </div>

    <!-- Dialog tambah/edit -->
    <UiDialog v-model:open="formOpen" :title="editing ? 'Edit tamu' : 'Tambah tamu'" size="sm" :persistent="saving">
      <form id="guest-form" class="space-y-4" @submit.prevent="submitForm(false)">
        <UiField label="Nama tamu" required :error="formErrors.name" hint="Tampil di sampul undangan, mis. “Bapak Budi & Keluarga”">
          <UiInput v-model="form.name" maxlength="100" :invalid="!!formErrors.name" />
        </UiField>
        <UiField label="No. WhatsApp" :error="formErrors.phone" hint="Opsional, untuk kirim undangan via WhatsApp">
          <UiInput v-model="form.phone" type="tel" inputmode="tel" placeholder="08xxxxxxxxxx" :invalid="!!formErrors.phone" />
        </UiField>
        <div class="grid grid-cols-3 gap-3">
          <UiField label="Grup" :error="formErrors.group_name" class="col-span-2">
            <UiInput v-model="form.group_name" :list="groupListId" placeholder="mis. Keluarga" />
            <datalist :id="groupListId">
              <option v-for="g in groups" :key="g" :value="g" />
            </datalist>
          </UiField>
          <UiField label="Jumlah" :error="formErrors.pax">
            <UiInput v-model="form.pax" type="number" min="1" max="20" />
          </UiField>
        </div>
      </form>
      <template #footer>
        <UiButton variant="ghost" :disabled="saving" @click="formOpen = false">Batal</UiButton>
        <UiButton v-if="!editing" variant="secondary" :disabled="saving" @click="submitForm(true)">Simpan & tambah lagi</UiButton>
        <UiButton type="submit" form="guest-form" :loading="saving">Simpan</UiButton>
      </template>
    </UiDialog>

    <!-- Dialog import -->
    <UiDialog v-model:open="importOpen" title="Import tamu dari Excel/CSV" :size="importStep === 2 ? 'lg' : 'md'" :persistent="importing">
      <ol class="mb-5 flex items-center gap-2 text-xs">
        <li v-for="(label, i) in ['Pilih file', 'Periksa', 'Import']" :key="label" class="flex items-center gap-2">
          <span
            class="flex size-6 items-center justify-center rounded-full font-semibold"
            :class="i + 1 <= importStep ? 'bg-accent text-accent-ink' : 'bg-surface-2 text-muted'"
          >{{ i + 1 }}</span>
          <span :class="i + 1 <= importStep ? 'font-medium text-ink' : 'text-muted'">{{ label }}</span>
          <span v-if="i < 2" class="h-px w-6 bg-line" />
        </li>
      </ol>

      <div v-if="importStep === 1" class="space-y-4">
        <div class="rounded-xl bg-surface-2 p-4 text-sm">
          <p class="mb-2 font-medium text-ink">Format kolom (baris pertama = judul kolom)</p>
          <div class="overflow-x-auto">
            <table class="w-full text-left text-xs">
              <thead>
                <tr class="text-muted">
                  <th class="py-1 pr-3 font-medium">Kolom</th>
                  <th class="py-1 pr-3 font-medium">Keterangan</th>
                  <th class="py-1 font-medium">Contoh</th>
                </tr>
              </thead>
              <tbody class="text-ink">
                <tr><td class="py-1 pr-3 font-mono">nama</td><td class="py-1 pr-3"><UiBadge tone="accent">wajib</UiBadge></td><td class="py-1">Bapak Budi &amp; Keluarga</td></tr>
                <tr><td class="py-1 pr-3 font-mono">no_hp</td><td class="py-1 pr-3 text-muted">opsional</td><td class="py-1">081234567890</td></tr>
                <tr><td class="py-1 pr-3 font-mono">grup</td><td class="py-1 pr-3 text-muted">opsional</td><td class="py-1">Keluarga</td></tr>
                <tr><td class="py-1 pr-3 font-mono">jumlah_tamu</td><td class="py-1 pr-3 text-muted">opsional (bawaan 1)</td><td class="py-1">2</td></tr>
              </tbody>
            </table>
          </div>
          <p class="mt-3 text-xs text-muted">Tamu duplikat (nama + no. HP sama) otomatis dilewati. Maks. {{ MAX_IMPORT_MB }} MB, 5.000 baris.</p>
          <div class="mt-3 flex flex-wrap gap-x-4 gap-y-1">
            <a :href="templateXlsxUrl" download class="inline-flex items-center gap-1.5 text-sm font-medium text-accent hover:underline">
              <UiIcon name="download" :size="15" /> Unduh template Excel
            </a>
            <a :href="templateUrl" download class="inline-flex items-center gap-1.5 text-sm font-medium text-accent hover:underline">
              <UiIcon name="download" :size="15" /> Template CSV
            </a>
          </div>
        </div>

        <input ref="fileInput" type="file" accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" class="hidden" @change="onImportFile" />
        <button
          type="button"
          class="flex w-full flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-line px-4 py-8 text-sm text-muted transition hover:border-accent hover:text-accent disabled:opacity-60"
          :disabled="importing"
          @click="fileInput?.click()"
        >
          <UiSpinner v-if="importing" :size="24" />
          <UiIcon v-else name="file" :size="26" />
          <span class="font-medium">{{ importing ? 'Memeriksa file…' : 'Pilih file .csv atau .xlsx' }}</span>
        </button>
        <p v-if="importError" class="flex items-start gap-1.5 text-sm text-danger"><UiIcon name="alert" :size="15" class="mt-0.5" /> {{ importError }}</p>
      </div>

      <div v-else-if="importPreview" class="space-y-4">
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <span class="mr-1 truncate text-muted"><UiIcon name="file" :size="14" class="inline" /> {{ importFile?.name }}</span>
          <UiBadge tone="success">{{ importPreview.summary.ok }} siap diimport</UiBadge>
          <UiBadge v-if="importPreview.summary.duplicate" tone="warning">{{ importPreview.summary.duplicate }} duplikat</UiBadge>
          <UiBadge v-if="importPreview.summary.error" tone="danger">{{ importPreview.summary.error }} error</UiBadge>
        </div>
        <p v-if="importOverQuota" class="rounded-xl bg-danger/10 px-3 py-2 text-sm text-danger">
          Jumlah tamu melebihi sisa kuota ({{ quotaLeft }}). Kurangi baris di file atau upgrade paket.
        </p>
        <p v-if="importPreview.summary.duplicate || importPreview.summary.error" class="text-xs text-muted">Baris duplikat dan error akan dilewati.</p>
        <div class="max-h-[50dvh] overflow-auto rounded-xl border border-line">
          <table class="w-full text-sm">
            <thead class="sticky top-0 bg-surface-2 text-left text-xs text-muted">
              <tr>
                <th class="px-3 py-2 font-medium">Baris</th>
                <th class="px-3 py-2 font-medium">Nama</th>
                <th class="px-3 py-2 font-medium">No. HP</th>
                <th class="px-3 py-2 font-medium">Grup</th>
                <th class="px-3 py-2 text-center font-medium">Jml</th>
                <th class="px-3 py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-line">
              <tr v-for="r in importPreview.rows" :key="r.row" :class="r.status === 'ok' ? '' : 'bg-surface-2/50'">
                <td class="px-3 py-2 text-muted">{{ r.row }}</td>
                <td class="px-3 py-2 text-ink">{{ r.name || '-' }}</td>
                <td class="px-3 py-2 whitespace-nowrap text-muted">{{ r.phone || '-' }}</td>
                <td class="px-3 py-2 text-muted">{{ r.group_name || '-' }}</td>
                <td class="px-3 py-2 text-center text-ink">{{ r.pax }}</td>
                <td class="px-3 py-2">
                  <UiBadge :tone="STATUS[r.status].tone">{{ STATUS[r.status].label }}</UiBadge>
                  <p v-if="r.error" class="mt-0.5 text-xs text-danger">{{ r.error }}</p>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="importError" class="flex items-start gap-1.5 text-sm text-danger"><UiIcon name="alert" :size="15" class="mt-0.5" /> {{ importError }}</p>
      </div>

      <template v-if="importStep === 2 && importPreview" #footer>
        <UiButton variant="ghost" :disabled="importing" class="mr-auto" @click="openImport">Pilih file lain</UiButton>
        <UiButton :disabled="!importPreview.summary.ok || importOverQuota" :loading="importing" @click="doImport">
          Import {{ importPreview.summary.ok }} tamu
        </UiButton>
      </template>
    </UiDialog>
  </div>
</template>
