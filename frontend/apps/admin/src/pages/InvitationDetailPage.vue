<script setup lang="ts">
import { api, formatDate, normalizeContent, type Invitation, type InvitationContent, type ThemeInfo } from '@undangan/shared'
import AttendanceReport from '@undangan/shared/components/AttendanceReport.vue'
import CustomDomainPanel from '@undangan/shared/components/CustomDomainPanel.vue'
import GiftConfirmations from '@undangan/shared/components/GiftConfirmations.vue'
import GuestManager from '@undangan/shared/components/GuestManager.vue'
import InvitationEditor from '@undangan/shared/components/InvitationEditor.vue'
import InvitationSettingsPanel from '@undangan/shared/components/InvitationSettingsPanel.vue'
import PreviewFrame from '@undangan/shared/components/PreviewFrame.vue'
import SubdomainField from '@undangan/shared/components/SubdomainField.vue'
import ThemePicker from '@undangan/shared/components/ThemePicker.vue'
import WishesManager from '@undangan/shared/components/WishesManager.vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import CopyButton from '../components/CopyButton.vue'
import UiBadge from '../components/UiBadge.vue'
import UiDialog from '../components/UiDialog.vue'
import UiSpinner from '../components/UiSpinner.vue'
import UiState from '../components/UiState.vue'
import UiTabs from '../components/UiTabs.vue'
import { confirmDialog } from '../lib/confirm'
import { EVENT_TYPE_LABEL, INVITATION_STATUS, SUBSCRIPTION_STATUS } from '../lib/labels'
import { toast } from '../lib/toast'
import { errMsg, formatNumber, itemsOf } from '../lib/util'

type Tab = 'data' | 'tema' | 'tamu' | 'kehadiran' | 'ucapan' | 'hadiah' | 'pengaturan'
const TABS: { value: Tab; label: string }[] = [
  { value: 'data', label: 'Data' },
  { value: 'tema', label: 'Tema' },
  { value: 'tamu', label: 'Tamu' },
  { value: 'kehadiran', label: 'Kehadiran' },
  { value: 'ucapan', label: 'Ucapan' },
  { value: 'hadiah', label: 'Hadiah' },
  { value: 'pengaturan', label: 'Pengaturan' },
]

/** Origin portal customer — stasiun check-in berjalan di sana. */
const PORTAL_URL = ((import.meta.env.VITE_PORTAL_URL as string | undefined) || 'http://localhost:5173').replace(/\/+$/, '')

const route = useRoute()
const router = useRouter()
const id = computed(() => String(route.params.id))

const inv = ref<Invitation | null>(null)
const loading = ref(true)
const error = ref('')

const tab = ref<Tab>(TABS.some((t) => t.value === route.query.tab) ? (route.query.tab as Tab) : 'data')
watch(tab, (t) => router.replace({ query: t === 'data' ? {} : { tab: t } }))

// ---- Konten (tab Data) ----
const content = ref<InvitationContent | null>(null)
const savedJson = ref('')
const dirty = computed(() => !!content.value && JSON.stringify(content.value) !== savedJson.value)
const saving = ref(false)
const reloadKey = ref(0)
const device = ref<'mobile' | 'desktop'>('mobile')
const previewDialog = ref(false)
const previewSrc = computed(() => api.url(`/invitations/${id.value}/preview`))

function setContent(raw: unknown) {
  const c = normalizeContent(raw)
  content.value = c
  savedJson.value = JSON.stringify(c)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Invitation>(`/invitations/${id.value}`)
    inv.value = res
    setContent(res.content)
  } catch (e) {
    error.value = errMsg(e)
  } finally {
    loading.value = false
  }
}

/** Perbarui metadata undangan tanpa menimpa konten yang sedang diedit. */
function applyMeta(next: Invitation) {
  inv.value = { ...next, content: inv.value?.content ?? next.content }
}
async function refreshMeta() {
  try {
    applyMeta(await api.get<Invitation>(`/invitations/${id.value}`))
  } catch {
    /* abaikan */
  }
}

async function save() {
  if (!content.value || !dirty.value || saving.value) return true
  saving.value = true
  const sentJson = JSON.stringify(content.value)
  try {
    const res = await api.patch<Invitation>(`/invitations/${id.value}`, { content: JSON.parse(sentJson) })
    const stillSame = content.value && JSON.stringify(content.value) === sentJson
    inv.value = res
    if (stillSame) setContent(res.content)
    else savedJson.value = sentJson
    reloadKey.value++
    toast.success('Data undangan disimpan.')
    return true
  } catch (e) {
    toast.error(errMsg(e))
    return false
  } finally {
    saving.value = false
  }
}

function onKey(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's' && tab.value === 'data') {
    e.preventDefault()
    save()
  }
}
function beforeUnload(e: BeforeUnloadEvent) {
  if (dirty.value) e.preventDefault()
}
onMounted(() => {
  load()
  window.addEventListener('keydown', onKey)
  window.addEventListener('beforeunload', beforeUnload)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('beforeunload', beforeUnload)
})
onBeforeRouteLeave(async () => {
  if (!dirty.value) return true
  return confirmDialog({
    title: 'Tinggalkan tanpa menyimpan?',
    message: 'Perubahan data undangan belum disimpan dan akan hilang.',
    confirmText: 'Tinggalkan',
    tone: 'danger',
  })
})

// ---- Publish / unpublish / hapus ----
const acting = ref<'publish' | 'unpublish' | 'delete' | null>(null)

async function publish() {
  const i = inv.value
  if (!i) return
  if (dirty.value && !(await save())) return
  acting.value = 'publish'
  try {
    applyMeta(await api.post<Invitation>(`/invitations/${i.id}/publish`))
    toast.success('Undangan diterbitkan.')
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    acting.value = null
  }
}

async function unpublish() {
  const i = inv.value
  if (!i) return
  const ok = await confirmDialog({
    title: 'Batalkan penerbitan?',
    message: 'Tamu tidak dapat membuka undangan sampai diterbitkan kembali.',
    details: ['Data, tamu, dan ucapan tetap tersimpan.', 'Subdomain dapat diubah customer lagi selama belum terbit.'],
    confirmText: 'Batalkan terbit',
    tone: 'danger',
  })
  if (!ok) return
  acting.value = 'unpublish'
  try {
    applyMeta(await api.post<Invitation>(`/invitations/${i.id}/unpublish`))
    toast.success('Undangan tidak lagi terbit.')
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    acting.value = null
  }
}

async function remove() {
  const i = inv.value
  if (!i) return
  const ok = await confirmDialog({
    title: `Hapus undangan “${i.title}”?`,
    message: `Undangan milik ${i.user_name} akan dihapus.`,
    details: [
      'Undangan langsung tidak dapat diakses tamu.',
      `${formatNumber(i.guest_count)} tamu dan seluruh ucapan ikut tersembunyi.`,
      'Penghapusan bersifat soft delete dan tercatat di audit log.',
    ],
    confirmText: 'Hapus undangan',
    tone: 'danger',
  })
  if (!ok) return
  acting.value = 'delete'
  try {
    await api.del(`/invitations/${i.id}`)
    savedJson.value = JSON.stringify(content.value) // lewati guard perubahan
    toast.success('Undangan dihapus.')
    router.push('/invitations')
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    acting.value = null
  }
}

// ---- Tema ----
const themes = ref<ThemeInfo[]>([])
const themesLoading = ref(false)
const themesError = ref('')
const themeBusy = ref(false)

async function loadThemes() {
  themesLoading.value = true
  themesError.value = ''
  try {
    themes.value = itemsOf(await api.get<ThemeInfo[] | { items: ThemeInfo[] }>('/themes'))
  } catch (e) {
    themesError.value = errMsg(e)
  } finally {
    themesLoading.value = false
  }
}
watch(tab, (t) => t === 'tema' && !themes.value.length && loadThemes(), { immediate: true })

async function selectTheme(slug: string) {
  const i = inv.value
  if (!i) return
  themeBusy.value = true
  try {
    const res = await api.put<Invitation | undefined>(`/invitations/${i.id}/theme`, { theme: slug })
    if (res && typeof res === 'object' && 'id' in res) applyMeta(res)
    else await refreshMeta()
    reloadKey.value++
    const name = themes.value.find((t) => t.slug === slug)?.name ?? slug
    toast.success(`Tema diganti ke ${name}. Tema kini terkunci untuk customer.`)
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    themeBusy.value = false
  }
}

function onSubdomainSaved(next: Invitation) {
  applyMeta(next)
  reloadKey.value++
}

const stationUrl = computed(() => (inv.value ? PORTAL_URL + (inv.value.settings?.checkin_station_path || `/checkin/${inv.value.id}`) : ''))

const status = computed(() => (inv.value ? INVITATION_STATUS[inv.value.status] : null))
const quotaPct = computed(() => {
  const q = inv.value?.quota
  return q && q.max_guests ? Math.min(100, Math.round((q.used_guests / q.max_guests) * 100)) : 0
})
</script>

<template>
  <div>
    <RouterLink to="/invitations" class="mb-3 inline-flex items-center gap-1 text-xs text-muted hover:text-ink">
      <AppIcon name="chevron-left" :size="14" /> Undangan
    </RouterLink>

    <UiState :loading="loading" :error="error" :empty="!inv" empty-title="Undangan tidak ditemukan" empty-icon="mail" @retry="load">
      <div v-if="inv">
        <!-- Header -->
        <div class="mb-4 flex flex-wrap items-start justify-between gap-4">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h1 class="truncate text-lg font-semibold tracking-tight">{{ inv.title }}</h1>
              <UiBadge v-if="status" :tone="status.tone">{{ status.label }}</UiBadge>
              <UiBadge v-if="content" tone="neutral" :dot="false">{{ EVENT_TYPE_LABEL[content.event_type] }}</UiBadge>
            </div>
            <p class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
              <span class="inline-flex items-center gap-1">
                <AppIcon name="user" :size="12" />
                <RouterLink :to="`/users/${inv.user_id}`" class="text-ink hover:text-accent">{{ inv.user_name }}</RouterLink>
                <span>· {{ inv.user_email }}</span>
              </span>
              <span v-if="inv.url" class="inline-flex items-center gap-0.5">
                <AppIcon name="globe" :size="12" />
                <a :href="inv.url" target="_blank" rel="noopener" class="ml-0.5 font-mono text-accent hover:underline">{{ inv.url.replace(/^https?:\/\//, '') }}</a>
                <CopyButton :text="inv.url" />
              </span>
              <span v-else class="inline-flex items-center gap-1"><AppIcon name="globe" :size="12" /> belum ada subdomain</span>
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <a v-if="inv.url && inv.status === 'published'" :href="inv.url" target="_blank" rel="noopener" class="btn btn-secondary">
              <AppIcon name="external" :size="14" /> Buka
            </a>
            <a v-if="inv.settings?.checkin_enabled" :href="stationUrl" target="_blank" rel="noopener" class="btn btn-secondary" title="Buka stasiun check-in di portal customer">
              <AppIcon name="shield" :size="14" /> Stasiun check-in
            </a>
            <button v-if="inv.status === 'published'" type="button" class="btn btn-secondary" :disabled="!!acting" @click="unpublish">
              <UiSpinner v-if="acting === 'unpublish'" :size="14" /><AppIcon v-else name="ban" :size="14" /> Batalkan terbit
            </button>
            <button v-else type="button" class="btn btn-primary" :disabled="!!acting || saving" @click="publish">
              <UiSpinner v-if="acting === 'publish'" :size="14" /><AppIcon v-else name="globe" :size="14" /> Terbitkan
            </button>
            <button type="button" class="btn btn-danger btn-icon" :disabled="!!acting" aria-label="Hapus undangan" title="Hapus undangan" @click="remove">
              <UiSpinner v-if="acting === 'delete'" :size="14" /><AppIcon v-else name="trash" :size="14" />
            </button>
          </div>
        </div>

        <!-- Ringkasan -->
        <div class="card mb-4 grid grid-cols-2 divide-line text-xs sm:grid-cols-4 sm:divide-x">
          <div class="px-4 py-2.5">
            <p class="text-muted">Tema</p>
            <p class="mt-0.5 flex items-center gap-1 text-[13px] font-medium">
              {{ inv.theme ?? 'belum dipilih' }}
              <AppIcon v-if="inv.theme_locked" name="key" :size="12" class="text-muted" title="Terkunci untuk customer" />
            </p>
          </div>
          <div class="px-4 py-2.5">
            <p class="text-muted">Tanggal acara</p>
            <p class="num mt-0.5 text-[13px] font-medium">{{ formatDate(inv.event_date) }}</p>
          </div>
          <div class="px-4 py-2.5">
            <p class="text-muted">Langganan pemilik</p>
            <p class="mt-0.5">
              <UiBadge v-if="inv.subscription_status" :tone="SUBSCRIPTION_STATUS[inv.subscription_status]?.tone">
                {{ SUBSCRIPTION_STATUS[inv.subscription_status]?.label }}
              </UiBadge>
              <span v-else class="text-[13px] text-muted">tidak ada</span>
            </p>
          </div>
          <div class="px-4 py-2.5">
            <p class="text-muted">Kuota tamu</p>
            <p class="num mt-0.5 text-[13px] font-medium">{{ formatNumber(inv.quota?.used_guests ?? inv.guest_count) }} / {{ formatNumber(inv.quota?.max_guests) }}</p>
            <div class="mt-1 h-1 overflow-hidden rounded bg-line">
              <div class="h-full rounded" :class="quotaPct >= 90 ? 'bg-warning' : 'bg-accent'" :style="{ width: `${quotaPct}%` }" />
            </div>
          </div>
        </div>

        <UiTabs v-model="tab" :tabs="TABS" aria-label="Bagian undangan" />

        <div class="pt-4">
          <!-- DATA -->
          <div v-if="tab === 'data' && content" class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_380px] 2xl:grid-cols-[minmax(0,1fr)_430px]">
            <div class="min-w-0">
              <div class="sticky top-14 z-10 -mx-1 mb-3 flex flex-wrap items-center gap-2 border-b border-line bg-canvas/90 px-1 py-2 backdrop-blur">
                <span v-if="dirty" class="inline-flex items-center gap-1.5 text-xs text-warning"><span class="size-1.5 rounded-full bg-warning" /> Perubahan belum disimpan</span>
                <span v-else class="inline-flex items-center gap-1.5 text-xs text-muted"><AppIcon name="check" :size="12" class="text-success" /> Tersimpan</span>
                <button type="button" class="btn btn-secondary btn-sm ml-auto lg:hidden" @click="previewDialog = true">
                  <AppIcon name="eye" :size="13" /> Pratinjau
                </button>
                <button v-if="dirty" type="button" class="btn btn-ghost btn-sm" :disabled="saving" @click="setContent(JSON.parse(savedJson))">Batalkan</button>
                <button type="button" class="btn btn-primary btn-sm" :class="!dirty && 'lg:ml-auto'" :disabled="!dirty || saving" title="Ctrl+S" @click="save">
                  <UiSpinner v-if="saving" :size="12" /> Simpan
                  <span class="kbd ml-1 hidden border-accent-ink/20 bg-transparent text-accent-ink/70 sm:inline">Ctrl S</span>
                </button>
              </div>
              <InvitationEditor v-model="content" :invitation-id="inv.id" />
            </div>

            <aside class="hidden lg:block" aria-label="Pratinjau undangan">
              <div class="sticky top-[4.5rem]">
                <div class="mb-2 flex items-center gap-2">
                  <p class="text-xs text-muted">Pratinjau</p>
                  <span v-if="dirty" class="text-[11px] text-warning">(simpan untuk memperbarui)</span>
                  <div class="ml-auto flex rounded-md bg-surface p-0.5 ring-1 ring-line" role="group" aria-label="Ukuran pratinjau">
                    <button
                      v-for="d in (['mobile', 'desktop'] as const)"
                      :key="d"
                      type="button"
                      class="grid h-6 w-7 place-items-center rounded"
                      :class="device === d ? 'bg-surface-2 text-ink' : 'text-muted hover:text-ink'"
                      :aria-pressed="device === d"
                      :aria-label="d === 'mobile' ? 'Ponsel' : 'Desktop'"
                      @click="device = d"
                    >
                      <AppIcon :name="d === 'mobile' ? 'phone' : 'monitor'" :size="13" />
                    </button>
                  </div>
                  <button type="button" class="btn btn-ghost btn-sm btn-icon h-7" aria-label="Muat ulang pratinjau" @click="reloadKey++">
                    <AppIcon name="refresh" :size="13" />
                  </button>
                  <a :href="previewSrc" target="_blank" rel="noopener" class="btn btn-ghost btn-sm btn-icon h-7" aria-label="Buka pratinjau di tab baru">
                    <AppIcon name="external" :size="13" />
                  </a>
                </div>
                <div class="h-[calc(100dvh-8rem)]">
                  <PreviewFrame :src="previewSrc" :reload-key="reloadKey" :device="device" />
                </div>
              </div>
            </aside>
          </div>

          <!-- TEMA -->
          <div v-else-if="tab === 'tema'" class="space-y-3">
            <div class="flex items-start gap-2 rounded-md border border-accent/25 bg-accent/5 px-3 py-2.5 text-xs text-ink/90">
              <AppIcon name="info" :size="14" class="mt-px text-accent" />
              <p>
                Admin dapat mengganti tema kapan saja. <b>Tema yang dipilih admin juga mengunci pilihan tema bagi customer</b> — customer tidak bisa menggantinya
                sendiri setelah ini.
              </p>
            </div>
            <UiState :loading="themesLoading" :error="themesError" :empty="!themes.length && themesLoading" @retry="loadThemes">
              <ThemePicker
                :themes="themes"
                :current="inv.theme"
                :locked="inv.theme_locked"
                :can-override-lock="true"
                :invitation-id="inv.id"
                :busy="themeBusy"
                @select="selectTheme"
              />
            </UiState>
          </div>

          <!-- TAMU -->
          <GuestManager v-else-if="tab === 'tamu'" :invitation="inv" @changed="refreshMeta" />

          <!-- KEHADIRAN -->
          <AttendanceReport v-else-if="tab === 'kehadiran'" :invitation="inv" />

          <!-- UCAPAN -->
          <WishesManager v-else-if="tab === 'ucapan'" :invitation-id="inv.id" />

          <!-- HADIAH -->
          <div v-else-if="tab === 'hadiah'" class="space-y-3">
            <div v-if="content && (!content.gift.enabled || !content.gift.confirmation_enabled)" class="flex items-start gap-2 rounded-md border border-warning/30 bg-warning/10 px-3 py-2.5 text-xs text-ink/90">
              <AppIcon name="info" :size="14" class="mt-px text-warning" />
              <p>Konfirmasi hadiah nonaktif untuk undangan ini. Aktifkan di tab Data → Amplop Digital → “Izinkan tamu mengirim bukti transfer/kado”.</p>
            </div>
            <GiftConfirmations :invitation-id="inv.id" />
          </div>

          <!-- PENGATURAN -->
          <div v-else-if="tab === 'pengaturan'" class="grid items-start gap-5 lg:grid-cols-3">
            <div class="min-w-0 space-y-5 lg:col-span-2">
              <section class="card" aria-labelledby="sub-title">
                <div class="card-header"><h2 id="sub-title" class="card-title">Subdomain</h2></div>
                <div class="p-4">
                  <SubdomainField :invitation="inv" :editable="true" @saved="onSubdomainSaved" />
                  <p class="hint mt-3">Mengubah subdomain undangan yang sudah terbit akan mengubah semua link tamu yang sudah dibagikan.</p>
                </div>
              </section>
              <section class="card" aria-labelledby="domain-title">
                <div class="card-header"><h2 id="domain-title" class="card-title">Domain sendiri</h2></div>
                <div class="p-4">
                  <CustomDomainPanel :invitation="inv" :is-admin="true" @saved="applyMeta" />
                </div>
              </section>
              <section class="card" aria-labelledby="access-title">
                <div class="card-header"><h2 id="access-title" class="card-title">Akses & check-in</h2></div>
                <div class="p-4">
                  <InvitationSettingsPanel :invitation="inv" :is-admin="true" :station-base-url="PORTAL_URL" @saved="applyMeta" />
                </div>
              </section>
            </div>
            <section class="card h-fit" aria-labelledby="info-title">
              <div class="card-header"><h2 id="info-title" class="card-title">Informasi</h2></div>
              <dl class="grid grid-cols-[110px_1fr] gap-x-3 gap-y-2 p-4 text-[13px]">
                <dt class="text-muted">ID</dt>
                <dd class="flex min-w-0 items-center gap-1">
                  <code class="truncate font-mono text-xs">{{ inv.id }}</code><CopyButton :text="inv.id" />
                </dd>
                <dt class="text-muted">Pemilik</dt>
                <dd class="min-w-0 truncate">
                  <RouterLink :to="`/users/${inv.user_id}`" class="hover:text-accent">{{ inv.user_name }}</RouterLink>
                </dd>
                <dt class="text-muted">Dibuat</dt>
                <dd class="num">{{ formatDate(inv.created_at, true) }}</dd>
                <dt class="text-muted">Diubah</dt>
                <dd class="num">{{ formatDate(inv.updated_at, true) }}</dd>
                <dt class="text-muted">Jumlah tamu</dt>
                <dd class="num">{{ formatNumber(inv.guest_count) }}</dd>
              </dl>
              <div class="border-t border-line p-4">
                <p class="text-[13px] font-medium text-danger">Zona berbahaya</p>
                <p class="mt-0.5 text-xs text-muted">Hapus undangan ini beserta akses tamunya.</p>
                <button type="button" class="btn btn-danger btn-sm mt-2.5" :disabled="!!acting" @click="remove">
                  <AppIcon name="trash" :size="13" /> Hapus undangan
                </button>
              </div>
            </section>
          </div>
        </div>
      </div>
    </UiState>

    <UiDialog :open="previewDialog" title="Pratinjau undangan" size="lg" @close="previewDialog = false">
      <div class="h-[70dvh]">
        <PreviewFrame v-if="previewDialog" :src="previewSrc" :reload-key="reloadKey" device="mobile" />
      </div>
    </UiDialog>
  </div>
</template>
