<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ApiError, api, formatDate } from '../api'
import { confirm } from '../composables/useConfirm'
import { toast } from '../composables/useToast'
import type { DnsRecord, Invitation } from '../types'
import { copyText, errorMessage } from '../utils/misc'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiIcon from './ui/UiIcon.vue'

const props = withDefaults(
  defineProps<{
    invitation: Invitation
    isAdmin: boolean
    /** Tujuan tombol "Upgrade paket" (portal customer), mis. "/paket". */
    upgradeTo?: string
  }>(),
  { upgradeTo: undefined },
)
const emit = defineEmits<{ saved: [inv: Invitation] }>()

const domain = computed(() => props.invitation.custom_domain ?? null)
const available = computed(() => props.invitation.settings?.custom_domain_available ?? false)
const locked = computed(() => !available.value && !props.isAdmin)

// ---------- Form hostname ----------
const editing = ref(false)
const hostname = ref('')
const hostError = ref('')
const saving = ref(false)
const removing = ref(false)

// ---------- Verifikasi ----------
const verifying = ref(false)
const dnsProblem = ref<{ message: string; items: string[] } | null>(null)
const lastCheckedAt = ref<Date | null>(null)

function resetState() {
  editing.value = false
  hostname.value = ''
  hostError.value = ''
  dnsProblem.value = null
  lastCheckedAt.value = null
}
watch(() => props.invitation.id, resetState)
watch(
  () => domain.value?.hostname,
  () => {
    dnsProblem.value = null
    lastCheckedAt.value = null
  },
)

/** Akhiran domain bawaan (mis. "undangan.id") dari URL undangan, untuk mencegah salah isi subdomain gratis. */
const baseDomain = computed(() => {
  const { url, subdomain } = props.invitation
  if (!url || !subdomain) return ''
  try {
    return new URL(url).hostname.replace(`${subdomain}.`, '')
  } catch {
    return ''
  }
})

function normalizeHost(v: string) {
  let h = v.trim().toLowerCase()
  h = h.replace(/^[a-z][a-z0-9+.-]*:\/\//, '') // skema
  h = h.replace(/[/?#].*$/, '') // path/query
  h = h.replace(/:\d+$/, '') // port
  h = h.replace(/\.+$/, '') // titik akhir
  return h
}

const HOST_RE = /^(?=.{4,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/

function validateHost(h: string): string {
  if (!h) return 'Masukkan nama domain, mis. www.rakadannadia.com'
  if (/\s/.test(h)) return 'Nama domain tidak boleh berisi spasi'
  if (!HOST_RE.test(h)) return 'Format domain tidak valid. Contoh: www.rakadannadia.com atau rakadannadia.com'
  const base = baseDomain.value
  if (base && (h === base || h.endsWith(`.${base}`))) return `Alamat .${base} sudah disediakan gratis lewat subdomain. Masukkan domain milik Anda sendiri.`
  return ''
}

const normalized = computed(() => normalizeHost(hostname.value))
const isApex = computed(() => normalized.value.split('.').length === 2)

function startEdit() {
  hostname.value = domain.value?.hostname ?? ''
  hostError.value = ''
  editing.value = true
}

function cancelEdit() {
  editing.value = false
  hostError.value = ''
}

async function save() {
  if (saving.value) return
  const h = normalized.value
  hostError.value = validateHost(h)
  if (hostError.value) return
  if (domain.value && h === domain.value.hostname) {
    editing.value = false
    return
  }
  if (domain.value) {
    const ok = await confirm({
      title: 'Ganti domain?',
      message: `Domain ${domain.value.hostname} akan dilepas dan diganti ${h}. Anda perlu mengatur ulang record DNS dan verifikasi dari awal.`,
      confirmText: 'Ya, ganti domain',
    })
    if (!ok) return
  }
  saving.value = true
  try {
    const inv = await api.put<Invitation>(`/invitations/${props.invitation.id}/custom-domain`, { hostname: h })
    editing.value = false
    hostname.value = ''
    emit('saved', inv)
    toast.success('Domain disimpan. Sekarang atur record DNS di registrar Anda.')
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      hostError.value = e.fields.hostname || 'Domain ini sudah dipakai undangan lain.'
    } else if (e instanceof ApiError && e.status === 422) {
      hostError.value = e.fields.hostname || e.message
    } else if (e instanceof ApiError && e.status === 403) {
      toast.error(e.code === 'feature_unavailable' ? 'Paket Anda belum mendukung domain sendiri. Upgrade paket untuk memakai fitur ini.' : e.message)
    } else {
      toast.error(e)
    }
  } finally {
    saving.value = false
  }
}

/** Pecah pesan backend menjadi butir bila berisi beberapa baris / dipisah titik koma. */
function splitProblems(e: ApiError): { message: string; items: string[] } {
  const fieldItems = Object.values(e.fields).filter(Boolean)
  if (fieldItems.length) return { message: e.message, items: fieldItems }
  const parts = e.message
    .split(/\r?\n|;\s+/)
    .map((s) => s.replace(/^\s*[-•*]\s*/, '').trim())
    .filter(Boolean)
  if (parts.length > 1) {
    // Kalimat pembuka yang diakhiri ":" dijadikan judul.
    const head = parts[0].endsWith(':') ? parts.shift()!.replace(/:$/, '') : 'DNS belum siap'
    return { message: head, items: parts }
  }
  return { message: e.message, items: [] }
}

async function verify() {
  if (verifying.value || !domain.value) return
  verifying.value = true
  dnsProblem.value = null
  try {
    const inv = await api.post<Invitation>(`/invitations/${props.invitation.id}/custom-domain/verify`)
    lastCheckedAt.value = new Date()
    emit('saved', inv)
    if (inv.custom_domain?.verified) toast.success('Domain terverifikasi! Sertifikat SSL sedang diterbitkan otomatis.')
    else dnsProblem.value = { message: 'DNS belum mengarah dengan benar. Coba lagi beberapa saat lagi.', items: [] }
  } catch (e) {
    lastCheckedAt.value = new Date()
    if (e instanceof ApiError && e.status === 422 && e.code === 'dns_not_ready') {
      dnsProblem.value = splitProblems(e)
    } else if (e instanceof ApiError && e.status === 403) {
      toast.error(e.code === 'feature_unavailable' ? 'Paket Anda belum mendukung domain sendiri.' : e.message)
    } else {
      toast.error(errorMessage(e))
    }
  } finally {
    verifying.value = false
  }
}

async function remove() {
  if (!domain.value || removing.value) return
  const ok = await confirm({
    title: 'Hapus domain sendiri?',
    message: `${domain.value.hostname} tidak akan lagi membuka undangan ini. Subdomain gratis tetap aktif. Record DNS di registrar bisa Anda hapus setelahnya.`,
    confirmText: 'Ya, hapus domain',
    danger: true,
  })
  if (!ok) return
  removing.value = true
  try {
    const inv = await api.del<Invitation>(`/invitations/${props.invitation.id}/custom-domain`)
    resetState()
    emit('saved', inv)
    toast.success('Domain sendiri dihapus')
  } catch (e) {
    toast.error(e)
  } finally {
    removing.value = false
  }
}

async function copy(text: string, label: string) {
  if (await copyText(text)) toast.success(`${label} disalin`)
  else toast.error('Gagal menyalin')
}

const showRecords = ref(false)
const recordsVisible = computed(() => !!domain.value && (!domain.value.verified || showRecords.value))
const recordKey = (r: DnsRecord, i: number) => `${r.type}-${r.name}-${i}`
const timeLabel = (d: Date) => d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
</script>

<template>
  <div class="space-y-4">
    <!-- Terkunci: paket tidak mendukung -->
    <div v-if="locked" class="flex flex-col gap-3 rounded-xl border border-dashed border-accent/40 bg-accent/5 p-4 sm:flex-row sm:items-center">
      <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent/12 text-accent"><UiIcon name="crown" :size="20" /></span>
      <div class="flex-1 text-sm">
        <p class="font-medium text-ink">Pakai domain sendiri, mis. www.rakadannadia.com</p>
        <p class="mt-0.5 text-muted">Fitur domain sendiri tersedia di paket yang mendukung. Subdomain gratis Anda tetap bisa dipakai.</p>
      </div>
      <RouterLink
        v-if="upgradeTo"
        :to="upgradeTo"
        class="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-xl bg-accent px-4 text-sm font-medium text-accent-ink shadow-sm transition hover:brightness-95"
      >
        Upgrade paket <UiIcon name="arrow-right" :size="15" />
      </RouterLink>
    </div>

    <template v-else>
      <p v-if="isAdmin && !available" class="flex items-start gap-2 rounded-xl bg-warning/10 px-3 py-2.5 text-xs text-ink">
        <UiIcon name="alert" :size="14" class="mt-0.5 shrink-0 text-warning" />
        Paket pemilik tidak mencakup domain sendiri. Sebagai admin Anda tetap bisa mengaturnya.
      </p>

      <!-- Penjelasan (belum ada domain) -->
      <div v-if="!domain" class="flex items-start gap-3 rounded-xl border border-line bg-surface-2 px-4 py-3 text-sm text-muted">
        <UiIcon name="info" :size="18" class="mt-0.5 shrink-0 text-accent" />
        <div class="space-y-1">
          <p>
            Buka undangan lewat alamat milik Anda sendiri, mis. <b class="font-medium text-ink">www.rakadannadia.com</b>. Domain dibeli sendiri di
            registrar seperti Niagahoster, Rumahweb, Domainesia, atau Cloudflare.
          </p>
          <p>Subdomain gratis tetap aktif, jadi link yang sudah dibagikan tetap berfungsi.</p>
        </div>
      </div>

      <!-- Domain terpasang -->
      <div v-if="domain && !editing" class="rounded-xl border border-line bg-surface p-3 sm:p-4">
        <div class="flex flex-wrap items-center gap-3">
          <span
            class="flex size-10 shrink-0 items-center justify-center rounded-full"
            :class="domain.verified ? 'bg-success/12 text-success' : 'bg-warning/15 text-warning'"
          >
            <UiIcon :name="domain.verified ? 'shield' : 'clock'" :size="19" />
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <a
                v-if="domain.verified"
                :href="`https://${domain.hostname}`"
                target="_blank"
                rel="noopener"
                class="min-w-0 truncate font-medium text-ink hover:text-accent hover:underline"
              >
                {{ domain.hostname }}
              </a>
              <span v-else class="min-w-0 truncate font-medium text-ink">{{ domain.hostname }}</span>
              <UiBadge v-if="domain.verified" tone="success"><UiIcon name="check" :size="11" /> Terverifikasi</UiBadge>
              <UiBadge v-else tone="warning"><UiIcon name="clock" :size="11" /> Menunggu DNS</UiBadge>
            </div>
            <p class="mt-0.5 text-xs text-muted">
              <template v-if="domain.verified">Aktif sejak {{ formatDate(domain.verified_at, true) }}</template>
              <template v-else>Tambahkan record DNS di bawah, lalu klik “Cek Verifikasi”.</template>
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <UiButton v-if="!domain.verified" size="sm" :loading="verifying" @click="verify">
              <UiIcon name="refresh" :size="14" /> Cek Verifikasi
            </UiButton>
            <UiButton v-else variant="secondary" size="sm" :href="`https://${domain.hostname}`" target="_blank" rel="noopener">
              <UiIcon name="external" :size="14" /> Buka
            </UiButton>
            <UiButton variant="ghost" size="sm" :disabled="removing" @click="startEdit"><UiIcon name="pencil" :size="14" /> Ganti</UiButton>
            <UiButton variant="ghost" size="sm" class="text-danger!" :loading="removing" @click="remove">
              <UiIcon name="trash" :size="14" /> Hapus
            </UiButton>
          </div>
        </div>

        <!-- Hasil cek verifikasi -->
        <div v-if="!domain.verified && dnsProblem" class="mt-3 flex items-start gap-2.5 rounded-xl border border-warning/30 bg-warning/10 px-3 py-2.5 text-sm text-ink" role="status">
          <UiIcon name="alert" :size="16" class="mt-0.5 shrink-0 text-warning" />
          <div class="min-w-0 flex-1">
            <p class="font-medium whitespace-pre-line">{{ dnsProblem.message }}</p>
            <ul v-if="dnsProblem.items.length" class="mt-1 list-disc space-y-0.5 pl-4 text-xs text-muted">
              <li v-for="(item, i) in dnsProblem.items" :key="i" class="break-words">{{ item }}</li>
            </ul>
            <p class="mt-1.5 text-xs text-muted">
              Perubahan DNS bisa butuh waktu hingga 24 jam untuk menyebar.
              <template v-if="lastCheckedAt">Terakhir dicek pukul {{ timeLabel(lastCheckedAt) }}.</template>
            </p>
          </div>
        </div>

        <p v-if="domain.verified" class="mt-3 flex items-start gap-2 rounded-xl bg-success/10 px-3 py-2.5 text-xs leading-relaxed text-ink">
          <UiIcon name="check-circle" :size="14" class="mt-0.5 shrink-0 text-success" />
          <span>
            Domain terhubung. Sertifikat SSL (https) terbit otomatis setelah terverifikasi — bila belum bisa dibuka, tunggu beberapa menit. Jangan hapus
            record DNS di registrar agar domain tetap tersambung.
          </span>
        </p>
      </div>

      <!-- Form hostname -->
      <form v-if="!domain || editing" class="space-y-2" novalidate @submit.prevent="save">
        <label for="custom-domain-host" class="block text-sm font-medium text-ink">{{ domain ? 'Domain baru' : 'Nama domain' }}</label>
        <div class="flex flex-col gap-2 sm:flex-row">
          <div
            class="flex min-w-0 flex-1 items-center overflow-hidden rounded-xl border bg-surface-2 transition focus-within:border-accent focus-within:bg-surface focus-within:ring-3 focus-within:ring-accent/15"
            :class="hostError ? 'border-danger' : 'border-line'"
          >
            <span class="pl-3 text-sm text-muted">https://</span>
            <input
              id="custom-domain-host"
              v-model="hostname"
              type="text"
              inputmode="url"
              autocapitalize="off"
              autocomplete="off"
              spellcheck="false"
              maxlength="253"
              placeholder="www.rakadannadia.com"
              :aria-invalid="!!hostError || undefined"
              aria-describedby="custom-domain-hint"
              class="h-10 min-w-0 flex-1 bg-transparent px-0.5 text-sm font-medium text-ink outline-none placeholder:font-normal placeholder:text-muted/70"
              @input="hostError = ''"
            />
          </div>
          <div class="flex gap-2">
            <UiButton v-if="editing" variant="ghost" :disabled="saving" @click="cancelEdit">Batal</UiButton>
            <UiButton type="submit" :loading="saving" :disabled="!hostname.trim()" class="flex-1 sm:flex-none">
              <UiIcon name="save" :size="16" /> Simpan domain
            </UiButton>
          </div>
        </div>
        <p v-if="hostError" class="flex items-start gap-1.5 text-xs text-danger" role="alert">
          <UiIcon name="alert" :size="14" class="mt-px shrink-0" /> {{ hostError }}
        </p>
        <p v-else id="custom-domain-hint" class="text-xs text-muted">
          Tanpa https:// atau garis miring.
          <template v-if="normalized && isApex">
            Domain utama (tanpa www) memerlukan record A — pastikan registrar Anda mendukungnya.
          </template>
          <template v-else>Contoh: www.rakadannadia.com (disarankan) atau rakadannadia.com.</template>
        </p>
      </form>

      <!-- Instruksi DNS -->
      <div v-if="domain && !editing" class="space-y-3">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <p class="text-xs font-medium tracking-wider text-muted uppercase">Record DNS di registrar</p>
          <button
            v-if="domain.verified"
            type="button"
            class="inline-flex items-center gap-1 text-xs font-medium text-accent hover:underline"
            :aria-expanded="showRecords"
            @click="showRecords = !showRecords"
          >
            {{ showRecords ? 'Sembunyikan' : 'Lihat record' }}
            <UiIcon :name="showRecords ? 'chevron-up' : 'chevron-down'" :size="14" />
          </button>
        </div>

        <template v-if="recordsVisible">
          <ol v-if="!domain.verified" class="list-decimal space-y-1 pl-5 text-xs leading-relaxed text-muted">
            <li>Masuk ke panel pengelolaan DNS di tempat Anda membeli domain (mis. Niagahoster, Rumahweb, Domainesia, Cloudflare).</li>
            <li>Tambahkan setiap record di bawah persis seperti tertulis. Hapus record lama dengan nama yang sama bila ada.</li>
            <li>Klik <b class="text-ink">Cek Verifikasi</b>. Bila memakai Cloudflare, set proxy ke <b class="text-ink">DNS only</b> (awan abu-abu).</li>
          </ol>

          <p v-if="!domain.dns.length" class="rounded-xl bg-surface-2 px-3 py-3 text-sm text-muted">Instruksi DNS belum tersedia. Muat ulang halaman beberapa saat lagi.</p>

          <div v-else class="overflow-hidden rounded-xl border border-line">
            <div class="hidden grid-cols-[4.5rem_minmax(0,1fr)_minmax(0,1.6fr)] gap-3 bg-surface-2 px-3 py-2 text-[11px] font-medium tracking-wider text-muted uppercase sm:grid">
              <span>Tipe</span><span>Nama / Host</span><span>Nilai / Target</span>
            </div>
            <div
              v-for="(r, i) in domain.dns"
              :key="recordKey(r, i)"
              class="grid gap-2 px-3 py-3 sm:grid-cols-[4.5rem_minmax(0,1fr)_minmax(0,1.6fr)] sm:gap-3"
              :class="i > 0 ? 'border-t border-line' : ''"
            >
              <div class="flex items-center gap-2 sm:block">
                <span class="text-[11px] tracking-wider text-muted uppercase sm:hidden">Tipe</span>
                <span class="inline-flex rounded-md bg-accent/12 px-2 py-0.5 font-mono text-xs font-semibold text-accent">{{ r.type }}</span>
              </div>
              <div class="min-w-0">
                <span class="mb-0.5 block text-[11px] tracking-wider text-muted uppercase sm:hidden">Nama / Host</span>
                <div class="flex items-center gap-1 rounded-lg bg-surface-2 py-1 pr-1 pl-2">
                  <code class="min-w-0 flex-1 font-mono text-xs break-all text-ink">{{ r.name }}</code>
                  <button type="button" class="shrink-0 rounded-md p-1.5 text-muted hover:bg-surface hover:text-ink" :aria-label="`Salin nama record ${r.type}`" @click="copy(r.name, 'Nama record')">
                    <UiIcon name="copy" :size="14" />
                  </button>
                </div>
              </div>
              <div class="min-w-0">
                <span class="mb-0.5 block text-[11px] tracking-wider text-muted uppercase sm:hidden">Nilai / Target</span>
                <div class="flex items-center gap-1 rounded-lg bg-surface-2 py-1 pr-1 pl-2">
                  <code class="min-w-0 flex-1 font-mono text-xs break-all text-ink">{{ r.value }}</code>
                  <button type="button" class="shrink-0 rounded-md p-1.5 text-muted hover:bg-surface hover:text-ink" :aria-label="`Salin nilai record ${r.type}`" @click="copy(r.value, 'Nilai record')">
                    <UiIcon name="copy" :size="14" />
                  </button>
                </div>
              </div>
              <p v-if="r.purpose" class="text-xs leading-relaxed text-muted sm:col-span-3">{{ r.purpose }}</p>
            </div>
          </div>

          <p v-if="!domain.verified" class="flex items-start gap-2 text-xs leading-relaxed text-muted">
            <UiIcon name="clock" :size="14" class="mt-0.5 shrink-0" />
            <span>
              Propagasi DNS bisa memakan waktu hingga <b class="text-ink">24 jam</b>, biasanya jauh lebih cepat. Sertifikat SSL (https) terbit otomatis
              setelah domain terverifikasi.
            </span>
          </p>
        </template>
      </div>
    </template>
  </div>
</template>
