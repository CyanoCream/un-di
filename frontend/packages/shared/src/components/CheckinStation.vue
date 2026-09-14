<script setup lang="ts">
import type QrScannerType from 'qr-scanner'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ApiError } from '../api'
import { createCheckinClient, isCheckinAuthError, rememberStationName, rememberedStationName } from '../checkin'
import { confirm } from '../composables/useConfirm'
import { toast } from '../composables/useToast'
import type { CheckinGuest, CheckinLog, CheckinResult, CheckinResultStatus, CheckinSummary } from '../types'
import { errorMessage } from '../utils/misc'
import UiButton from './ui/UiButton.vue'
import UiIcon, { type IconName } from './ui/UiIcon.vue'
import UiSpinner from './ui/UiSpinner.vue'
import UiStepper from './ui/UiStepper.vue'

const props = withDefaults(
  defineProps<{
    invitationId: string
    /** Link login pemilik (opsional), ditampilkan di layar PIN. */
    loginUrl?: string
  }>(),
  { loginUrl: undefined },
)

const SUMMARY_MS = 20_000
const RESULT_MS = 3_000
const SAME_SCAN_MS = 4_000
const SOUND_KEY = 'checkin:sound'

let client = createCheckinClient(props.invitationId)

// ======================================================================
// Autentikasi
// ======================================================================
type Phase = 'loading' | 'pin' | 'main' | 'error'
const phase = ref<Phase>('loading')
const fatalError = ref('')
const mode = ref<'owner' | 'station'>('station')
const title = ref('')
const summary = ref<CheckinSummary | null>(null)

const pin = ref('')
const stationName = ref(rememberedStationName())
const pinError = ref('')
const loggingIn = ref(false)
const showPin = ref(false)

async function init() {
  phase.value = 'loading'
  fatalError.value = ''
  try {
    await loadSummary()
    mode.value = client.hasToken() ? 'station' : 'owner'
    phase.value = 'main'
  } catch (e) {
    if (isCheckinAuthError(e)) {
      phase.value = 'pin'
    } else {
      fatalError.value = e instanceof ApiError && e.status === 404 ? 'Undangan tidak ditemukan. Periksa kembali link stasiun check-in.' : errorMessage(e)
      phase.value = 'error'
    }
  }
}

function toPin(message = 'Sesi stasiun berakhir. Masukkan PIN lagi.') {
  stopCamera()
  closeResult()
  client.logout()
  pin.value = ''
  pinError.value = message
  phase.value = 'pin'
}

/** Tangani error umum; kembalikan true bila sudah ditangani sebagai error autentikasi. */
function handleError(e: unknown, silent = false) {
  if (isCheckinAuthError(e)) {
    toPin()
    return true
  }
  if (!silent) toast.error(e)
  return false
}

function onPinInput(e: Event) {
  const el = e.target as HTMLInputElement
  const digits = el.value.replace(/\D/g, '').slice(0, 8)
  el.value = digits
  pin.value = digits
  pinError.value = ''
}

async function login() {
  unlockAudio()
  if (!/^\d{6,8}$/.test(pin.value)) {
    pinError.value = 'PIN terdiri dari 6–8 digit angka'
    return
  }
  loggingIn.value = true
  pinError.value = ''
  try {
    const s = await client.login(pin.value, stationName.value)
    rememberStationName(stationName.value)
    title.value = s.invitation?.title ?? title.value
    pin.value = ''
    mode.value = 'station'
    await loadSummary().catch(() => undefined)
    phase.value = 'main'
  } catch (e) {
    if (e instanceof ApiError && e.status === 429) pinError.value = 'Terlalu banyak percobaan. Tunggu beberapa menit lalu coba lagi.'
    else if (e instanceof ApiError && e.status === 401) pinError.value = e.message && e.code !== 'unknown' ? e.message : 'PIN salah'
    else pinError.value = errorMessage(e)
  } finally {
    loggingIn.value = false
  }
}

async function logout() {
  const ok = await confirm({
    title: 'Keluar dari stasiun?',
    message: 'Perangkat ini perlu memasukkan PIN lagi untuk melanjutkan check-in.',
    confirmText: 'Keluar',
    danger: true,
  })
  if (!ok) return
  toPin('')
}

// ======================================================================
// Ringkasan
// ======================================================================
async function loadSummary() {
  const s = await client.summary()
  summary.value = s
  if (s.invitation?.title) title.value = s.invitation.title
  return s
}

async function refreshSummary() {
  try {
    await loadSummary()
  } catch (e) {
    handleError(e, true)
  }
}

const counters = computed(() => {
  const s = summary.value?.summary
  const guests = s?.checked_in_guests ?? 0
  const invited = s?.invited_guests ?? 0
  return {
    guests,
    invited,
    pax: s?.checked_in_pax ?? 0,
    invitedPax: s?.invited_pax ?? 0,
    pct: invited > 0 ? Math.min(100, Math.round((guests / invited) * 100)) : 0,
  }
})
const checkinDisabled = computed(() => summary.value?.checkin_enabled === false)

// ======================================================================
// Tab
// ======================================================================
type Tab = 'scan' | 'cari' | 'riwayat'
const tab = ref<Tab>('scan')
const TABS: { value: Tab; label: string; icon: IconName }[] = [
  { value: 'scan', label: 'Scan / Kode', icon: 'scan' },
  { value: 'cari', label: 'Cari nama', icon: 'search' },
  { value: 'riwayat', label: 'Riwayat', icon: 'clock' },
]

// ======================================================================
// Suara & getar
// ======================================================================
const soundOn = ref(readSound())
function readSound() {
  try {
    return localStorage.getItem(SOUND_KEY) !== '0'
  } catch {
    return true
  }
}
watch(soundOn, (v) => {
  try {
    localStorage.setItem(SOUND_KEY, v ? '1' : '0')
  } catch {
    /* abaikan */
  }
})

let audioCtx: AudioContext | null = null
function unlockAudio() {
  try {
    const Ctx = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!Ctx) return
    audioCtx ??= new Ctx()
    if (audioCtx.state === 'suspended') void audioCtx.resume()
  } catch {
    audioCtx = null
  }
}

function tone(freq: number, start: number, dur: number, type: OscillatorType = 'sine') {
  if (!audioCtx) return
  const t0 = audioCtx.currentTime + start
  const osc = audioCtx.createOscillator()
  const gain = audioCtx.createGain()
  osc.type = type
  osc.frequency.value = freq
  gain.gain.setValueAtTime(0.0001, t0)
  gain.gain.exponentialRampToValueAtTime(0.25, t0 + 0.01)
  gain.gain.exponentialRampToValueAtTime(0.0001, t0 + dur)
  osc.connect(gain).connect(audioCtx.destination)
  osc.start(t0)
  osc.stop(t0 + dur + 0.02)
}

function feedback(status: CheckinResultStatus) {
  if (soundOn.value) {
    unlockAudio()
    if (status === 'checked_in') {
      tone(880, 0, 0.12)
      tone(1320, 0.14, 0.2)
    } else if (status === 'already') {
      tone(560, 0, 0.14, 'triangle')
      tone(560, 0.22, 0.14, 'triangle')
    } else if (status === 'not_found') {
      tone(170, 0, 0.5, 'square')
    } else {
      tone(320, 0, 0.25, 'triangle')
    }
  }
  const pattern = { checked_in: [80], already: [80, 60, 80], not_found: [300, 100, 300], disabled: [150] }[status]
  try {
    navigator.vibrate?.(pattern)
  } catch {
    /* tidak didukung */
  }
}

// ======================================================================
// Hasil scan (overlay)
// ======================================================================
const result = ref<CheckinResult | null>(null)
const resultKey = ref(0)
let resultTimer: ReturnType<typeof setTimeout> | undefined
let lastScan = { text: '', at: 0 }

const RESULT_UI: Record<CheckinResultStatus, { bg: string; icon: IconName; title: string }> = {
  checked_in: { bg: 'bg-success', icon: 'check-circle', title: 'Silakan masuk' },
  already: { bg: 'bg-warning', icon: 'alert', title: 'Sudah check-in' },
  not_found: { bg: 'bg-danger', icon: 'x-circle', title: 'Tidak terdaftar' },
  disabled: { bg: 'bg-muted', icon: 'lock', title: 'Check-in nonaktif' },
}

function showResult(r: CheckinResult) {
  result.value = r
  resultKey.value++
  feedback(r.status)
  if (resultTimer) clearTimeout(resultTimer)
  resultTimer = setTimeout(closeResult, RESULT_MS)
}

function closeResult() {
  if (resultTimer) clearTimeout(resultTimer)
  resultTimer = undefined
  if (result.value) lastScan.at = Date.now() // QR yang masih di depan kamera tidak langsung terbaca ulang
  result.value = null
}

function time(iso: string | null | undefined) {
  if (!iso) return ''
  return new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit' }).format(new Date(iso))
}

const submitting = ref(false)

async function submit(code: string, pax?: number) {
  if (submitting.value) return null
  submitting.value = true
  try {
    const r = await client.scan(code, pax)
    showResult(r)
    if (r.guest) patchGuest(r.guest)
    if (r.status === 'checked_in') void refreshSummary()
    if (tab.value === 'riwayat') void loadRecent()
    return r
  } catch (e) {
    handleError(e)
    return null
  } finally {
    submitting.value = false
  }
}

// ======================================================================
// Kamera (qr-scanner)
// ======================================================================
type CamState = 'idle' | 'starting' | 'on' | 'error'
const videoEl = ref<HTMLVideoElement | null>(null)
const camState = ref<CamState>('idle')
const camError = ref<'denied' | 'notfound' | 'busy' | 'insecure' | 'other' | ''>('')
const hasFlash = ref(false)
const flashOn = ref(false)
const secure = typeof window !== 'undefined' && window.isSecureContext
let scanner: QrScannerType | null = null
let wantCamera = false
let wakeLock: WakeLockSentinel | null = null

function classifyCamError(e: unknown): typeof camError.value {
  const name = e instanceof DOMException || e instanceof Error ? e.name : ''
  const msg = String(e instanceof Error ? e.message : e).toLowerCase()
  if (name === 'NotAllowedError' || name === 'PermissionDeniedError' || msg.includes('permission') || msg.includes('denied')) return 'denied'
  if (name === 'NotFoundError' || name === 'OverconstrainedError' || msg.includes('not found')) return 'notfound'
  if (name === 'NotReadableError' || name === 'TrackStartError') return 'busy'
  return 'other'
}

function onDecode(text: string) {
  const value = text.trim()
  if (!value || result.value || submitting.value) return
  const now = Date.now()
  if (value === lastScan.text && now - lastScan.at < SAME_SCAN_MS) return
  lastScan = { text: value, at: now }
  void submit(value)
}

async function startCamera() {
  unlockAudio()
  wantCamera = true
  camError.value = ''
  if (!secure || !navigator.mediaDevices?.getUserMedia) {
    camState.value = 'error'
    camError.value = 'insecure'
    return
  }
  camState.value = 'starting'
  try {
    const { default: QrScanner } = await import('qr-scanner')
    await nextTick()
    if (!videoEl.value || !wantCamera) {
      camState.value = 'idle'
      return
    }
    scanner ??= new QrScanner(videoEl.value, (r) => onDecode(r.data), {
      preferredCamera: 'environment',
      maxScansPerSecond: 8,
      returnDetailedScanResult: true,
      onDecodeError: () => {
        /* frame tanpa QR — abaikan */
      },
    })
    await scanner.start()
    if (!wantCamera) {
      scanner.stop()
      return
    }
    camState.value = 'on'
    flashOn.value = false
    hasFlash.value = await scanner.hasFlash().catch(() => false)
    void requestWakeLock()
  } catch (e) {
    scanner?.destroy()
    scanner = null
    camState.value = 'error'
    camError.value = classifyCamError(e)
  }
}

function stopCamera(keepIntent = false) {
  if (!keepIntent) wantCamera = false
  scanner?.stop()
  if (camState.value !== 'error') camState.value = 'idle'
  flashOn.value = false
  void releaseWakeLock()
}

async function toggleFlash() {
  if (!scanner) return
  try {
    await scanner.toggleFlash()
    flashOn.value = scanner.isFlashOn()
  } catch {
    toast.error('Senter tidak dapat dinyalakan di perangkat ini')
  }
}

async function requestWakeLock() {
  try {
    if ('wakeLock' in navigator && document.visibilityState === 'visible') wakeLock = await navigator.wakeLock.request('screen')
  } catch {
    wakeLock = null
  }
}
async function releaseWakeLock() {
  try {
    await wakeLock?.release()
  } catch {
    /* abaikan */
  }
  wakeLock = null
}

watch(tab, (t, prev) => {
  if (prev === 'scan' && t !== 'scan' && camState.value === 'on') stopCamera(true)
  if (t === 'scan' && wantCamera && camState.value === 'idle') void startCamera()
  if (t === 'riwayat') void loadRecent()
  if (t === 'cari' && q.value.trim()) void runSearch()
})

// ---------- Kode manual ----------
const manualCode = ref('')
function onCodeInput(e: Event) {
  const el = e.target as HTMLInputElement
  const v = el.value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 6)
  el.value = v
  manualCode.value = v
}
async function submitCode() {
  if (manualCode.value.length !== 6) return
  unlockAudio()
  const r = await submit(manualCode.value)
  if (r) manualCode.value = ''
}

// ======================================================================
// Cari nama
// ======================================================================
const q = ref('')
const results = ref<CheckinGuest[]>([])
const searching = ref(false)
const searched = ref(false)
const expandedId = ref<string | null>(null)
const pax = ref(1)
let searchSeq = 0
let searchTimer: ReturnType<typeof setTimeout> | undefined

async function runSearch() {
  const term = q.value.trim()
  const my = ++searchSeq
  if (!term) {
    results.value = []
    searched.value = false
    searching.value = false
    return
  }
  searching.value = true
  try {
    const res = await client.searchGuests(term)
    if (my !== searchSeq) return
    results.value = res.items ?? []
    searched.value = true
  } catch (e) {
    if (my === searchSeq) handleError(e)
  } finally {
    if (my === searchSeq) searching.value = false
  }
}

watch(q, () => {
  if (searchTimer) clearTimeout(searchTimer)
  expandedId.value = null
  searchTimer = setTimeout(runSearch, 300)
})

function patchGuest(g: CheckinGuest) {
  const i = results.value.findIndex((x) => x.id === g.id)
  if (i >= 0) results.value[i] = { ...results.value[i]!, ...g }
}

function startGuestCheckin(g: CheckinGuest) {
  unlockAudio()
  if ((g.pax || 1) > 1) {
    expandedId.value = expandedId.value === g.id ? null : g.id
    pax.value = g.pax
  } else {
    void submit(g.code, 1)
  }
}

async function confirmGuestCheckin(g: CheckinGuest) {
  const r = await submit(g.code, pax.value)
  if (r) expandedId.value = null
}

// ======================================================================
// Riwayat
// ======================================================================
const recent = ref<CheckinLog[]>([])
const recentLoading = ref(false)
const recentLoaded = ref(false)

async function loadRecent() {
  recentLoading.value = true
  try {
    const res = await client.recent()
    recent.value = res.items ?? []
    recentLoaded.value = true
  } catch (e) {
    handleError(e, recentLoaded.value)
  } finally {
    recentLoading.value = false
  }
}

const LOG_UI: Record<CheckinResultStatus, { icon: IconName; cls: string; label: string }> = {
  checked_in: { icon: 'check-circle', cls: 'bg-success/12 text-success', label: 'Masuk' },
  already: { icon: 'alert', cls: 'bg-warning/15 text-warning', label: 'Sudah check-in' },
  not_found: { icon: 'x-circle', cls: 'bg-danger/12 text-danger', label: 'Ditolak' },
  disabled: { icon: 'lock', cls: 'bg-surface-2 text-muted', label: 'Nonaktif' },
}

function logTime(iso: string) {
  return new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' }).format(new Date(iso))
}

// ======================================================================
// Siklus hidup
// ======================================================================
let pollTimer: ReturnType<typeof setInterval> | undefined

function onVisibility() {
  if (document.visibilityState === 'visible' && phase.value === 'main') {
    void refreshSummary()
    if (camState.value === 'on') void requestWakeLock()
  }
}

onMounted(() => {
  void init()
  pollTimer = setInterval(() => {
    if (phase.value !== 'main' || document.visibilityState !== 'visible') return
    void refreshSummary()
    if (tab.value === 'riwayat') void loadRecent()
  }, SUMMARY_MS)
  document.addEventListener('visibilitychange', onVisibility)
})

watch(
  () => props.invitationId,
  () => {
    stopCamera()
    scanner?.destroy()
    scanner = null
    client = createCheckinClient(props.invitationId)
    summary.value = null
    title.value = ''
    results.value = []
    recent.value = []
    void init()
  },
)

watch(phase, (p) => {
  if (p !== 'main') stopCamera()
})

onBeforeUnmount(() => {
  wantCamera = false
  scanner?.destroy()
  scanner = null
  void releaseWakeLock()
  if (pollTimer) clearInterval(pollTimer)
  if (resultTimer) clearTimeout(resultTimer)
  if (searchTimer) clearTimeout(searchTimer)
  document.removeEventListener('visibilitychange', onVisibility)
  void audioCtx?.close().catch(() => undefined)
})
</script>

<template>
  <div class="min-h-dvh bg-surface-2 text-ink">
    <!-- ===== Memuat ===== -->
    <div v-if="phase === 'loading'" class="flex min-h-dvh flex-col items-center justify-center gap-3 text-sm text-muted">
      <UiSpinner :size="28" /> Membuka stasiun check-in…
    </div>

    <!-- ===== Error fatal ===== -->
    <div v-else-if="phase === 'error'" class="flex min-h-dvh items-center justify-center p-6">
      <div class="w-full max-w-sm rounded-3xl border border-line bg-surface p-6 text-center shadow-sm">
        <span class="mx-auto flex size-14 items-center justify-center rounded-full bg-danger/10 text-danger"><UiIcon name="alert" :size="26" /></span>
        <p class="mt-4 font-semibold text-ink">Stasiun tidak dapat dibuka</p>
        <p class="mt-1 text-sm text-muted">{{ fatalError }}</p>
        <UiButton class="mt-5 h-12!" block @click="init"><UiIcon name="refresh" :size="16" /> Coba lagi</UiButton>
      </div>
    </div>

    <!-- ===== PIN ===== -->
    <div v-else-if="phase === 'pin'" class="flex min-h-dvh items-center justify-center p-4 pt-[max(1rem,env(safe-area-inset-top))]">
      <form class="w-full max-w-sm rounded-3xl border border-line bg-surface p-6 shadow-sm" @submit.prevent="login">
        <div class="text-center">
          <span class="mx-auto flex size-16 items-center justify-center rounded-2xl bg-accent/12 text-accent"><UiIcon name="qr" :size="30" /></span>
          <h1 class="mt-4 text-xl font-semibold text-ink">Stasiun Check-in Tamu</h1>
          <p v-if="title" class="mt-0.5 font-medium text-ink">{{ title }}</p>
          <p class="mt-1 text-sm text-muted">Masukkan PIN dari pemilik acara untuk mulai menerima tamu.</p>
        </div>

        <label class="mt-6 block">
          <span class="mb-1.5 block text-sm font-medium text-ink">Nama meja / penerima</span>
          <input
            v-model="stationName"
            maxlength="40"
            placeholder="mis. Meja 1"
            autocomplete="off"
            class="h-12 w-full rounded-xl border border-line bg-surface-2 px-3 text-base text-ink outline-none focus:border-accent focus:bg-surface focus:ring-3 focus:ring-accent/15"
          />
        </label>

        <label class="mt-4 block">
          <span class="mb-1.5 block text-sm font-medium text-ink">PIN</span>
          <span class="relative block">
            <input
              :value="pin"
              :type="showPin ? 'text' : 'password'"
              inputmode="numeric"
              autocomplete="one-time-code"
              maxlength="8"
              placeholder="••••••"
              autofocus
              :aria-invalid="!!pinError || undefined"
              class="h-14 w-full rounded-xl border bg-surface-2 px-12 text-center font-mono text-2xl tracking-[0.4em] text-ink outline-none focus:border-accent focus:bg-surface focus:ring-3 focus:ring-accent/15"
              :class="pinError ? 'border-danger' : 'border-line'"
              @input="onPinInput"
            />
            <button
              type="button"
              class="absolute top-1/2 right-2 -translate-y-1/2 rounded-lg p-2 text-muted hover:text-ink"
              :aria-label="showPin ? 'Sembunyikan PIN' : 'Tampilkan PIN'"
              @click="showPin = !showPin"
            >
              <UiIcon :name="showPin ? 'eye-off' : 'eye'" :size="18" />
            </button>
          </span>
        </label>
        <p v-if="pinError" class="mt-2 flex items-start gap-1.5 text-sm text-danger" role="alert"><UiIcon name="alert" :size="15" class="mt-0.5" /> {{ pinError }}</p>

        <UiButton type="submit" block class="mt-5 h-13! text-base!" :loading="loggingIn" :disabled="pin.length < 6">
          <UiIcon name="key" :size="18" /> Buka stasiun
        </UiButton>

        <p v-if="loginUrl" class="mt-4 text-center text-xs text-muted">
          Pemilik acara?
          <a :href="loginUrl" class="font-medium text-accent hover:underline">Masuk ke akun</a>
          untuk membuka tanpa PIN.
        </p>
      </form>
    </div>

    <!-- ===== Utama ===== -->
    <div v-else class="mx-auto flex min-h-dvh max-w-xl flex-col pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <!-- Header -->
      <header class="sticky top-0 z-20 border-b border-line bg-surface/95 px-4 pt-[max(0.75rem,env(safe-area-inset-top))] pb-3 backdrop-blur">
        <div class="flex items-center gap-2">
          <div class="min-w-0 flex-1">
            <p class="truncate text-base font-semibold text-ink">{{ title || 'Check-in tamu' }}</p>
            <p class="flex items-center gap-1 truncate text-xs text-muted">
              <UiIcon :name="mode === 'owner' ? 'user' : 'qr'" :size="12" />
              {{ mode === 'owner' ? 'Mode pemilik' : stationName || 'Stasiun check-in' }}
            </p>
          </div>
          <button
            type="button"
            class="flex size-10 items-center justify-center rounded-xl text-muted hover:bg-surface-2 hover:text-ink"
            :aria-label="soundOn ? 'Matikan suara' : 'Nyalakan suara'"
            :aria-pressed="soundOn"
            @click="(soundOn = !soundOn), unlockAudio()"
          >
            <UiIcon :name="soundOn ? 'volume-2' : 'volume-x'" :size="19" />
          </button>
          <button
            v-if="mode === 'station'"
            type="button"
            class="flex size-10 items-center justify-center rounded-xl text-muted hover:bg-danger/10 hover:text-danger"
            aria-label="Keluar dari stasiun"
            @click="logout"
          >
            <UiIcon name="logout" :size="19" />
          </button>
        </div>

        <div class="mt-3 grid grid-cols-2 gap-2">
          <div class="rounded-xl bg-surface-2 px-3 py-2">
            <p class="text-[11px] text-muted">Tamu datang</p>
            <p class="text-lg leading-tight font-semibold text-ink tabular-nums">
              {{ counters.guests }}<span class="text-sm font-normal text-muted"> / {{ counters.invited }}</span>
            </p>
          </div>
          <div class="rounded-xl bg-surface-2 px-3 py-2">
            <p class="text-[11px] text-muted">Orang datang</p>
            <p class="text-lg leading-tight font-semibold text-ink tabular-nums">
              {{ counters.pax }}<span class="text-sm font-normal text-muted"> / {{ counters.invitedPax }}</span>
            </p>
          </div>
        </div>
        <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-surface-2" role="progressbar" :aria-valuenow="counters.pct" aria-valuemin="0" aria-valuemax="100">
          <div class="h-full rounded-full bg-accent transition-all duration-500" :style="{ width: `${counters.pct}%` }" />
        </div>
      </header>

      <p v-if="checkinDisabled" class="mx-4 mt-3 flex items-start gap-2 rounded-xl border border-warning/30 bg-warning/10 px-3 py-2.5 text-sm text-ink">
        <UiIcon name="alert" :size="16" class="mt-0.5 text-warning" /> Check-in sedang dinonaktifkan oleh pemilik acara. Scan tidak akan dicatat.
      </p>

      <main class="flex-1 px-4 pt-4">
        <!-- ----- Scan / kode ----- -->
        <section v-show="tab === 'scan'" class="space-y-4">
          <div class="relative mx-auto aspect-square w-full max-w-md overflow-hidden rounded-3xl bg-ink">
            <video ref="videoEl" class="absolute inset-0 size-full object-cover" :class="camState === 'on' ? '' : 'opacity-0'" muted playsinline />

            <!-- Bingkai bidik -->
            <div v-if="camState === 'on'" class="pointer-events-none absolute inset-0 flex items-center justify-center">
              <div class="relative size-[68%]">
                <span class="absolute top-0 left-0 size-10 rounded-tl-2xl border-t-4 border-l-4 border-surface" />
                <span class="absolute top-0 right-0 size-10 rounded-tr-2xl border-t-4 border-r-4 border-surface" />
                <span class="absolute bottom-0 left-0 size-10 rounded-bl-2xl border-b-4 border-l-4 border-surface" />
                <span class="absolute right-0 bottom-0 size-10 rounded-br-2xl border-r-4 border-b-4 border-surface" />
                <span class="scanline absolute inset-x-3 h-0.5 rounded-full bg-accent" />
              </div>
              <p class="absolute inset-x-0 bottom-3 text-center text-xs font-medium text-surface drop-shadow">
                <UiSpinner v-if="submitting" :size="12" class="mr-1 inline" />Arahkan kamera ke QR di undangan tamu
              </p>
            </div>

            <!-- Idle / memulai -->
            <div v-if="camState === 'idle' || camState === 'starting'" class="absolute inset-0 flex flex-col items-center justify-center gap-4 p-6 text-center">
              <span class="flex size-20 items-center justify-center rounded-full bg-surface/10 text-surface"><UiIcon name="qr" :size="40" /></span>
              <button
                type="button"
                class="flex h-16 w-full max-w-64 items-center justify-center gap-2 rounded-2xl bg-accent text-lg font-semibold text-accent-ink shadow-lg transition active:scale-[0.98] disabled:opacity-70"
                :disabled="camState === 'starting'"
                @click="startCamera"
              >
                <UiSpinner v-if="camState === 'starting'" :size="20" />
                <UiIcon v-else name="camera" :size="22" />
                {{ camState === 'starting' ? 'Membuka kamera…' : 'Scan QR' }}
              </button>
              <p class="text-xs text-surface/70">Kamera belakang akan digunakan</p>
            </div>

            <!-- Error kamera -->
            <div v-if="camState === 'error'" class="absolute inset-0 flex flex-col items-center justify-center gap-3 overflow-y-auto p-5 text-center text-surface">
              <UiIcon name="camera-off" :size="36" />
              <template v-if="camError === 'denied'">
                <p class="font-semibold">Akses kamera ditolak</p>
                <p class="text-xs leading-relaxed text-surface/80">
                  Ketuk ikon gembok / "Aa" di kolom alamat browser → Izin situs → izinkan <b>Kamera</b>, lalu muat ulang halaman. Sementara itu gunakan
                  input kode di bawah.
                </p>
              </template>
              <template v-else-if="camError === 'insecure'">
                <p class="font-semibold">Kamera butuh koneksi HTTPS</p>
                <p class="text-xs leading-relaxed text-surface/80">Buka link stasiun yang diawali <b>https://</b>. Anda tetap bisa check-in dengan kode atau cari nama.</p>
              </template>
              <template v-else-if="camError === 'notfound'">
                <p class="font-semibold">Kamera tidak ditemukan</p>
                <p class="text-xs text-surface/80">Gunakan input kode undangan atau cari nama tamu.</p>
              </template>
              <template v-else-if="camError === 'busy'">
                <p class="font-semibold">Kamera sedang dipakai aplikasi lain</p>
                <p class="text-xs text-surface/80">Tutup aplikasi kamera/video call lain lalu coba lagi.</p>
              </template>
              <p v-else class="font-semibold">Kamera tidak dapat dibuka</p>
              <button type="button" class="mt-1 inline-flex h-11 items-center gap-2 rounded-xl bg-surface px-4 text-sm font-medium text-ink" @click="startCamera">
                <UiIcon name="refresh" :size="16" /> Coba lagi
              </button>
            </div>

            <!-- Kontrol kamera -->
            <div v-if="camState === 'on'" class="absolute top-3 right-3 left-3 flex justify-between">
              <button
                type="button"
                class="flex h-11 items-center gap-1.5 rounded-xl bg-ink/60 px-3 text-sm font-medium text-surface backdrop-blur"
                @click="stopCamera()"
              >
                <UiIcon name="x" :size="16" /> Stop
              </button>
              <button
                v-if="hasFlash"
                type="button"
                class="flex size-11 items-center justify-center rounded-xl backdrop-blur"
                :class="flashOn ? 'bg-warning text-ink' : 'bg-ink/60 text-surface'"
                :aria-pressed="flashOn"
                aria-label="Senter"
                @click="toggleFlash"
              >
                <UiIcon name="zap" :size="19" />
              </button>
            </div>
          </div>

          <form class="rounded-2xl border border-line bg-surface p-4" @submit.prevent="submitCode">
            <label for="checkin-code" class="mb-2 block text-sm font-medium text-ink">Kode undangan</label>
            <div class="flex gap-2">
              <input
                id="checkin-code"
                :value="manualCode"
                maxlength="6"
                autocomplete="off"
                autocapitalize="characters"
                spellcheck="false"
                placeholder="ABC123"
                class="h-14 min-w-0 flex-1 rounded-xl border border-line bg-surface-2 px-3 text-center font-mono text-2xl tracking-[0.3em] text-ink uppercase outline-none placeholder:text-muted/50 focus:border-accent focus:bg-surface focus:ring-3 focus:ring-accent/15"
                @input="onCodeInput"
              />
              <button
                type="submit"
                class="flex h-14 shrink-0 items-center gap-2 rounded-xl bg-accent px-5 text-base font-semibold text-accent-ink transition active:scale-[0.98] disabled:opacity-50"
                :disabled="manualCode.length !== 6 || submitting"
              >
                <UiSpinner v-if="submitting" :size="18" /> Cek
              </button>
            </div>
            <p class="mt-2 text-xs text-muted">6 karakter yang tertera di tiket tamu, di bawah QR.</p>
          </form>
        </section>

        <!-- ----- Cari nama ----- -->
        <section v-show="tab === 'cari'" class="space-y-3">
          <div class="relative">
            <UiIcon name="search" :size="20" class="pointer-events-none absolute top-1/2 left-4 -translate-y-1/2 text-muted" />
            <input
              v-model="q"
              type="search"
              placeholder="Ketik nama tamu…"
              autocomplete="off"
              class="h-14 w-full rounded-2xl border border-line bg-surface pr-4 pl-12 text-base text-ink outline-none focus:border-accent focus:ring-3 focus:ring-accent/15"
            />
            <UiSpinner v-if="searching" :size="18" class="absolute top-1/2 right-4 -translate-y-1/2 text-muted" />
          </div>

          <p v-if="!q.trim()" class="px-2 py-8 text-center text-sm text-muted">Cari tamu yang tidak membawa QR atau lupa kodenya.</p>
          <p v-else-if="searched && !results.length && !searching" class="rounded-2xl border border-danger/25 bg-danger/5 px-4 py-6 text-center text-sm text-ink">
            <UiIcon name="x-circle" :size="22" class="mx-auto mb-1 text-danger" />
            Tidak ada tamu bernama “{{ q.trim() }}” di daftar.
          </p>

          <ul v-else class="space-y-2">
            <li v-for="g in results" :key="g.id" class="rounded-2xl border border-line bg-surface p-3">
              <div class="flex items-center gap-3">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-base font-semibold text-ink">{{ g.name }}</p>
                  <p class="truncate text-xs text-muted">
                    {{ g.pax }} orang<template v-if="g.group_name"> · {{ g.group_name }}</template> · <span class="font-mono">{{ g.code }}</span>
                  </p>
                  <p v-if="g.checked_in_at" class="mt-0.5 flex items-center gap-1 text-xs font-medium text-success">
                    <UiIcon name="check-circle" :size="13" /> Datang {{ time(g.checked_in_at) }} · {{ g.checked_in_pax ?? g.pax }} orang
                  </p>
                </div>
                <span v-if="g.checked_in_at" class="rounded-xl bg-success/12 px-3 py-2 text-sm font-medium text-success">Sudah</span>
                <button
                  v-else
                  type="button"
                  class="flex h-12 shrink-0 items-center gap-1.5 rounded-xl bg-accent px-4 text-sm font-semibold text-accent-ink transition active:scale-[0.98] disabled:opacity-50"
                  :disabled="submitting"
                  @click="startGuestCheckin(g)"
                >
                  <UiIcon name="user-check" :size="17" /> Check-in
                </button>
              </div>
              <div v-if="expandedId === g.id && !g.checked_in_at" class="mt-3 flex flex-col items-center gap-3 border-t border-line pt-3">
                <p class="text-sm text-muted">Berapa orang yang datang? (maks. {{ g.pax }})</p>
                <UiStepper v-model="pax" :min="1" :max="Math.max(1, g.pax)" size="lg" label="Jumlah orang yang datang" />
                <button
                  type="button"
                  class="flex h-12 w-full items-center justify-center gap-2 rounded-xl bg-success text-base font-semibold text-surface transition active:scale-[0.98] disabled:opacity-50"
                  :disabled="submitting"
                  @click="confirmGuestCheckin(g)"
                >
                  <UiSpinner v-if="submitting" :size="18" /><UiIcon v-else name="check" :size="18" /> Konfirmasi {{ pax }} orang
                </button>
              </div>
            </li>
          </ul>
        </section>

        <!-- ----- Riwayat ----- -->
        <section v-show="tab === 'riwayat'" class="space-y-2">
          <div class="flex items-center justify-between px-1">
            <p class="text-sm font-medium text-ink">20 aktivitas terakhir</p>
            <button type="button" class="flex size-10 items-center justify-center rounded-xl text-muted hover:bg-surface hover:text-ink" aria-label="Muat ulang riwayat" @click="loadRecent">
              <UiIcon name="refresh" :size="17" :class="recentLoading ? 'animate-spin' : ''" />
            </button>
          </div>
          <div v-if="!recentLoaded" class="flex items-center justify-center gap-2 py-12 text-sm text-muted"><UiSpinner /> Memuat riwayat…</div>
          <p v-else-if="!recent.length" class="py-12 text-center text-sm text-muted">Belum ada aktivitas check-in.</p>
          <ul v-else class="divide-y divide-line overflow-hidden rounded-2xl border border-line bg-surface">
            <li v-for="log in recent" :key="log.id" class="flex items-center gap-3 px-3 py-2.5">
              <span class="flex size-9 shrink-0 items-center justify-center rounded-full" :class="LOG_UI[log.result]?.cls ?? 'bg-surface-2 text-muted'">
                <UiIcon :name="LOG_UI[log.result]?.icon ?? 'info'" :size="17" />
              </span>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-medium text-ink">{{ log.guest_name || log.input || '-' }}</p>
                <p class="truncate text-xs text-muted">
                  {{ LOG_UI[log.result]?.label ?? log.result }}<template v-if="log.pax && log.result === 'checked_in'"> · {{ log.pax }} orang</template><template v-if="log.station"> · {{ log.station }}</template>
                </p>
              </div>
              <span class="shrink-0 text-xs text-muted tabular-nums">{{ logTime(log.created_at) }}</span>
            </li>
          </ul>
        </section>
      </main>

      <!-- Navigasi bawah -->
      <nav class="fixed inset-x-0 bottom-0 z-20 border-t border-line bg-surface/95 pb-[env(safe-area-inset-bottom)] backdrop-blur" aria-label="Menu stasiun">
        <div class="mx-auto grid max-w-xl grid-cols-3">
          <button
            v-for="t in TABS"
            :key="t.value"
            type="button"
            class="flex h-16 flex-col items-center justify-center gap-1 text-xs font-medium transition"
            :class="tab === t.value ? 'text-accent' : 'text-muted'"
            :aria-current="tab === t.value ? 'page' : undefined"
            @click="tab = t.value"
          >
            <UiIcon :name="t.icon" :size="22" />
            {{ t.label }}
          </button>
        </div>
      </nav>
    </div>

    <!-- ===== Overlay hasil ===== -->
    <Transition enter-active-class="transition duration-150 ease-out" enter-from-class="opacity-0 scale-95" leave-active-class="transition duration-150 ease-in" leave-to-class="opacity-0">
      <div
        v-if="result"
        :key="resultKey"
        class="fixed inset-0 z-[70] flex flex-col items-center justify-center p-6 pt-[max(1.5rem,env(safe-area-inset-top))] text-center text-surface"
        :class="RESULT_UI[result.status]?.bg ?? 'bg-muted'"
        role="alertdialog"
        aria-live="assertive"
        @click="closeResult"
      >
        <UiIcon :name="RESULT_UI[result.status]?.icon ?? 'info'" :size="110" :stroke-width="1.6" />
        <p class="mt-4 text-3xl font-extrabold tracking-tight uppercase">{{ RESULT_UI[result.status]?.title ?? result.status }}</p>

        <template v-if="result.guest && result.status !== 'not_found'">
          <p class="mt-4 text-3xl leading-tight font-bold break-words">{{ result.guest.name }}</p>
          <p v-if="result.guest.group_name" class="mt-1 text-lg opacity-90">{{ result.guest.group_name }}</p>
          <p class="mt-3 rounded-full bg-ink/15 px-4 py-1.5 text-lg font-semibold">
            Diundang {{ result.guest.pax }} orang<template v-if="result.status === 'checked_in' && result.guest.checked_in_pax"> · masuk {{ result.guest.checked_in_pax }}</template>
          </p>
          <p v-if="result.status === 'already' && result.guest.checked_in_at" class="mt-3 text-xl font-semibold">
            Check-in pukul {{ time(result.guest.checked_in_at) }}<template v-if="result.guest.checked_in_pax"> · {{ result.guest.checked_in_pax }} orang</template>
          </p>
        </template>
        <p v-else-if="result.status === 'not_found'" class="mt-3 text-xl font-semibold">Tidak terdaftar — tidak dapat masuk</p>

        <p v-if="result.message" class="mt-3 max-w-sm text-base opacity-90">{{ result.message }}</p>

        <div class="absolute inset-x-0 bottom-0 pb-[max(1.25rem,env(safe-area-inset-bottom))]">
          <p class="text-sm opacity-80">Ketuk untuk menutup</p>
          <div class="mx-auto mt-2 h-1 w-40 overflow-hidden rounded-full bg-ink/20">
            <div class="countdown h-full bg-surface" :style="{ animationDuration: `${RESULT_MS}ms` }" />
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.scanline {
  animation: scanline 2.2s ease-in-out infinite alternate;
}
@keyframes scanline {
  from {
    top: 8%;
  }
  to {
    top: 92%;
  }
}
.countdown {
  width: 100%;
  animation-name: countdown;
  animation-timing-function: linear;
  animation-fill-mode: forwards;
}
@keyframes countdown {
  from {
    width: 100%;
  }
  to {
    width: 0%;
  }
}
@media (prefers-reduced-motion: reduce) {
  .scanline {
    animation: none;
    top: 50%;
  }
}
</style>
