<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ApiError, api } from '../api'
import { toast } from '../composables/useToast'
import type { AccessMode, Invitation, InvitationSettings } from '../types'
import { copyText } from '../utils/misc'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiIcon from './ui/UiIcon.vue'
import UiSwitch from './ui/UiSwitch.vue'

const props = withDefaults(
  defineProps<{
    invitation: Invitation
    isAdmin: boolean
    /** Tujuan tombol "Upgrade paket" (portal customer), mis. "/paket". */
    upgradeTo?: string
    /** Origin portal customer untuk link stasiun check-in. Bawaan: origin halaman ini. */
    stationBaseUrl?: string
  }>(),
  { upgradeTo: undefined, stationBaseUrl: undefined },
)
const emit = defineEmits<{ saved: [inv: Invitation] }>()

const settings = computed<InvitationSettings>(() => ({
  access_mode: props.invitation.settings?.access_mode ?? 'public',
  checkin_enabled: props.invitation.settings?.checkin_enabled ?? false,
  checkin_pin_set: props.invitation.settings?.checkin_pin_set ?? false,
  checkin_available: props.invitation.settings?.checkin_available ?? false,
  checkin_station_path: props.invitation.settings?.checkin_station_path || `/checkin/${props.invitation.id}`,
  custom_domain_available: props.invitation.settings?.custom_domain_available ?? false,
}))

// ---------- Form ----------
const accessMode = ref<AccessMode>(settings.value.access_mode)
const checkinEnabled = ref(settings.value.checkin_enabled)
const pin = ref('')
const showPin = ref(false)
const changingPin = ref(false)
const pinError = ref('')
const saving = ref(false)

function reset() {
  accessMode.value = settings.value.access_mode
  checkinEnabled.value = settings.value.checkin_enabled
  pin.value = ''
  pinError.value = ''
  changingPin.value = false
  showPin.value = false
}
watch(() => [props.invitation.id, props.invitation.settings] as const, reset)

const locked = computed(() => !settings.value.checkin_available && !props.isAdmin)
const dirty = computed(
  () => accessMode.value !== settings.value.access_mode || checkinEnabled.value !== settings.value.checkin_enabled || pin.value !== '',
)

function onPinInput(e: Event) {
  const el = e.target as HTMLInputElement
  const digits = el.value.replace(/\D/g, '').slice(0, 8)
  el.value = digits
  pin.value = digits
  pinError.value = ''
}

function cancelPinChange() {
  changingPin.value = false
  pin.value = ''
  pinError.value = ''
}

function validate() {
  pinError.value = ''
  if (pin.value && !/^\d{6,8}$/.test(pin.value)) {
    pinError.value = 'PIN harus 6–8 digit angka'
    return false
  }
  if (checkinEnabled.value && !settings.value.checkin_pin_set && !pin.value) {
    pinError.value = 'Buat PIN terlebih dahulu untuk mengaktifkan check-in'
    return false
  }
  return true
}

async function save() {
  if (!validate()) return
  saving.value = true
  try {
    const body: { access_mode: AccessMode; checkin_enabled: boolean; checkin_pin?: string } = {
      access_mode: accessMode.value,
      checkin_enabled: locked.value ? settings.value.checkin_enabled : checkinEnabled.value,
    }
    if (pin.value && !locked.value) body.checkin_pin = pin.value
    const updated = await api.put<Invitation>(`/invitations/${props.invitation.id}/settings`, body)
    emit('saved', updated)
    toast.success(body.checkin_pin && settings.value.checkin_pin_set ? 'Pengaturan disimpan. Stasiun yang memakai PIN lama perlu masuk ulang.' : 'Pengaturan disimpan')
    // Bila parent tidak memperbarui prop (atau respons sama), tetap bersihkan field PIN.
    pin.value = ''
    changingPin.value = false
  } catch (e) {
    if (e instanceof ApiError && e.fields.checkin_pin) pinError.value = e.fields.checkin_pin
    else toast.error(e)
  } finally {
    saving.value = false
  }
}

// ---------- Stasiun ----------
const stationUrl = computed(() => {
  const origin = (props.stationBaseUrl || window.location.origin).replace(/\/+$/, '')
  return origin + settings.value.checkin_station_path
})

async function copyStation() {
  if (await copyText(stationUrl.value)) toast.success('Link stasiun check-in disalin')
  else toast.error('Gagal menyalin link')
}

const ACCESS_OPTIONS: { value: AccessMode; title: string; desc: string; icon: 'globe' | 'shield' }[] = [
  {
    value: 'public',
    title: 'Publik',
    desc: 'Siapa pun dengan link bisa membuka; nama dari link ?to= ditampilkan tanpa verifikasi.',
    icon: 'globe',
  },
  {
    value: 'guest_only',
    title: 'Khusus tamu terdaftar',
    desc: 'Hanya link pribadi (/nama-tamu) atau kode undangan yang bisa membuka undangan, RSVP & kirim hadiah.',
    icon: 'shield',
  },
]
</script>

<template>
  <div class="space-y-5">
    <!-- Akses undangan -->
    <section class="rounded-2xl border border-line bg-surface p-4 sm:p-5">
      <div class="mb-4">
        <h3 class="flex items-center gap-2 text-base font-semibold text-ink"><UiIcon name="lock" :size="17" class="text-accent" /> Akses undangan</h3>
        <p class="mt-0.5 text-sm text-muted">Tentukan siapa yang boleh membuka undangan.</p>
      </div>
      <div class="grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label="Mode akses undangan">
        <label
          v-for="o in ACCESS_OPTIONS"
          :key="o.value"
          class="relative flex cursor-pointer gap-3 rounded-xl border p-4 transition"
          :class="accessMode === o.value ? 'border-accent bg-accent/8 ring-3 ring-accent/15' : 'border-line bg-surface hover:bg-surface-2'"
        >
          <input v-model="accessMode" type="radio" name="access_mode" :value="o.value" class="sr-only" />
          <span
            class="flex size-9 shrink-0 items-center justify-center rounded-lg"
            :class="accessMode === o.value ? 'bg-accent text-accent-ink' : 'bg-surface-2 text-muted'"
          >
            <UiIcon :name="o.icon" :size="18" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="flex items-center gap-2 text-sm font-semibold text-ink">
              {{ o.title }}
              <UiBadge v-if="settings.access_mode === o.value" tone="success">Aktif</UiBadge>
            </span>
            <span class="mt-1 block text-xs leading-relaxed text-muted">{{ o.desc }}</span>
          </span>
          <span
            class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border-2"
            :class="accessMode === o.value ? 'border-accent' : 'border-line'"
            aria-hidden="true"
          >
            <span v-if="accessMode === o.value" class="size-2.5 rounded-full bg-accent" />
          </span>
        </label>
      </div>
      <p v-if="accessMode === 'guest_only'" class="mt-3 flex items-start gap-2 rounded-xl bg-surface-2 px-3 py-2.5 text-xs leading-relaxed text-muted">
        <UiIcon name="info" :size="14" class="mt-0.5 text-accent" />
        <span>
          Tamu tanpa link pribadi akan melihat halaman gerbang untuk memasukkan kode undangan. Link nama bisa ditebak — untuk acara yang benar-benar
          tertutup, aktifkan juga check-in di venue.
        </span>
      </p>
    </section>

    <!-- Check-in venue -->
    <section class="rounded-2xl border border-line bg-surface p-4 sm:p-5">
      <div class="mb-4 flex items-start justify-between gap-3">
        <div>
          <h3 class="flex items-center gap-2 text-base font-semibold text-ink"><UiIcon name="qr" :size="17" class="text-accent" /> Check-in QR di venue</h3>
          <p class="mt-0.5 text-sm text-muted">Penerima tamu memindai QR tiket tamu atau mengetik kode undangan di pintu masuk.</p>
        </div>
        <UiBadge v-if="settings.checkin_enabled" tone="success">Aktif</UiBadge>
        <UiBadge v-else-if="locked"><UiIcon name="lock" :size="11" /> Terkunci</UiBadge>
      </div>

      <!-- Terkunci (paket tidak mendukung) -->
      <div v-if="locked" class="flex flex-col gap-3 rounded-xl border border-dashed border-accent/40 bg-accent/5 p-4 sm:flex-row sm:items-center">
        <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-accent/12 text-accent"><UiIcon name="crown" :size="20" /></span>
        <p class="flex-1 text-sm text-ink">Fitur check-in tersedia di paket yang mendukung. Upgrade paket untuk mengaktifkan tiket QR & stasiun penerima tamu.</p>
        <RouterLink
          v-if="upgradeTo"
          :to="upgradeTo"
          class="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-xl bg-accent px-4 text-sm font-medium text-accent-ink shadow-sm transition hover:brightness-95"
        >
          Upgrade paket <UiIcon name="arrow-right" :size="15" />
        </RouterLink>
      </div>

      <div v-else class="space-y-4">
        <p v-if="isAdmin && !settings.checkin_available" class="flex items-start gap-2 rounded-xl bg-warning/10 px-3 py-2.5 text-xs text-ink">
          <UiIcon name="alert" :size="14" class="mt-0.5 text-warning" /> Paket pemilik tidak mencakup check-in. Sebagai admin Anda tetap bisa mengaktifkannya.
        </p>

        <UiSwitch
          v-model="checkinEnabled"
          label="Aktifkan check-in"
          description="Halaman tamu personal menampilkan tiket berisi QR, kode undangan, dan jumlah orang."
        />

        <!-- PIN -->
        <div v-if="checkinEnabled" class="rounded-xl border border-line bg-surface-2/50 p-3 sm:p-4">
          <div v-if="settings.checkin_pin_set && !changingPin" class="flex flex-wrap items-center gap-3">
            <span class="flex size-9 items-center justify-center rounded-lg bg-success/12 text-success"><UiIcon name="key" :size="17" /></span>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-ink">PIN sudah diatur</p>
              <p class="text-xs text-muted">PIN dipakai penerima tamu untuk membuka stasiun check-in.</p>
            </div>
            <UiButton variant="secondary" size="sm" @click="changingPin = true"><UiIcon name="pencil" :size="14" /> Ganti PIN</UiButton>
          </div>
          <UiField
            v-else
            :label="settings.checkin_pin_set ? 'PIN baru' : 'PIN stasiun check-in'"
            :required="!settings.checkin_pin_set"
            :error="pinError"
            :hint="settings.checkin_pin_set ? 'Stasiun yang sudah masuk dengan PIN lama akan keluar otomatis.' : '6–8 digit angka. Bagikan hanya ke penerima tamu.'"
          >
            <div class="flex gap-2">
              <div class="relative min-w-0 flex-1 sm:max-w-60">
                <input
                  :value="pin"
                  :type="showPin ? 'text' : 'password'"
                  inputmode="numeric"
                  autocomplete="new-password"
                  maxlength="8"
                  placeholder="••••••"
                  :aria-invalid="!!pinError || undefined"
                  class="h-10 w-full rounded-xl border bg-surface pr-10 pl-3 font-mono text-base tracking-[0.3em] text-ink outline-none focus:border-accent focus:ring-3 focus:ring-accent/15"
                  :class="pinError ? 'border-danger' : 'border-line'"
                  @input="onPinInput"
                />
                <button
                  type="button"
                  class="absolute top-1/2 right-1.5 -translate-y-1/2 rounded-lg p-1.5 text-muted hover:text-ink"
                  :aria-label="showPin ? 'Sembunyikan PIN' : 'Tampilkan PIN'"
                  @click="showPin = !showPin"
                >
                  <UiIcon :name="showPin ? 'eye-off' : 'eye'" :size="16" />
                </button>
              </div>
              <UiButton v-if="settings.checkin_pin_set" variant="ghost" @click="cancelPinChange">Batal</UiButton>
            </div>
          </UiField>
        </div>
        <p v-else-if="settings.checkin_enabled" class="text-xs text-warning">Simpan untuk menonaktifkan check-in. Tiket QR tidak lagi tampil di undangan.</p>

        <!-- Link stasiun -->
        <div v-if="settings.checkin_enabled" class="space-y-3 border-t border-line pt-4">
          <p class="text-xs font-medium tracking-wider text-muted uppercase">Link stasiun penerima tamu</p>
          <div class="flex items-center gap-2 rounded-xl border border-line bg-surface-2 p-1.5 pl-3">
            <span class="min-w-0 flex-1 truncate font-mono text-xs text-ink sm:text-sm">{{ stationUrl }}</span>
            <UiButton variant="secondary" size="sm" @click="copyStation"><UiIcon name="copy" :size="14" /> Salin</UiButton>
            <UiButton variant="secondary" size="sm" :href="stationUrl" target="_blank" rel="noopener" aria-label="Buka stasiun">
              <UiIcon name="external" :size="14" />
            </UiButton>
          </div>
          <ol class="list-decimal space-y-1 pl-5 text-xs leading-relaxed text-muted">
            <li>Kirim link di atas ke penerima tamu. Buka di HP (Chrome/Safari) — tidak perlu akun.</li>
            <li>Masukkan PIN dan nama meja (mis. "Meja 1"). Satu HP bisa dipakai beberapa jam tanpa login ulang.</li>
            <li>Tekan <b class="text-ink">Scan QR</b> dan arahkan kamera ke QR di undangan tamu, atau ketik kode 6 karakter / cari nama.</li>
            <li>Hijau = boleh masuk, kuning = sudah check-in sebelumnya, merah = tidak terdaftar.</li>
          </ol>
        </div>
        <p v-else-if="checkinEnabled" class="text-xs text-muted">Link stasiun penerima tamu muncul setelah pengaturan disimpan.</p>
      </div>
    </section>

    <div class="flex flex-wrap items-center justify-end gap-2">
      <span v-if="dirty" class="mr-auto inline-flex items-center gap-1.5 text-xs text-warning"><span class="size-1.5 rounded-full bg-warning" /> Perubahan belum disimpan</span>
      <UiButton v-if="dirty" variant="ghost" :disabled="saving" @click="reset">Batalkan</UiButton>
      <UiButton :disabled="!dirty" :loading="saving" @click="save"><UiIcon name="save" :size="16" /> Simpan pengaturan</UiButton>
    </div>
  </div>
</template>
