<script setup lang="ts">
import { computed, nextTick, reactive, ref, useId, watch } from 'vue'
import {
  BANK_SUGGESTIONS,
  EVENT_NAME_SUGGESTIONS,
  EWALLET_SUGGESTIONS,
  RELIGIONS,
  fillShareTemplate,
  newEvent,
  normalizeContent,
} from '../content'
import type { EventType, InvitationContent, Religion, Timezone } from '../types'
import { defaultWhatsappTemplate } from '../utils/misc'
import EditorSection from './editor/EditorSection.vue'
import PersonFields from './editor/PersonFields.vue'
import ImageUpload from './ImageUpload.vue'
import MusicPicker from './MusicPicker.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiIcon, { type IconName } from './ui/UiIcon.vue'
import UiInput from './ui/UiInput.vue'
import UiSelect from './ui/UiSelect.vue'
import UiSwitch from './ui/UiSwitch.vue'
import UiTextarea from './ui/UiTextarea.vue'

const props = defineProps<{ modelValue: InvitationContent; invitationId: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: InvitationContent] }>()

// ---------- State lokal: salinan prop; setiap perubahan dikirim sebagai objek baru ----------
function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T
}

const local = ref<InvitationContent>(normalizeContent(clone(props.modelValue)))
let lastEmitted: InvitationContent | null = null
let skipEmit = false

watch(
  () => props.modelValue,
  (v) => {
    if (v === lastEmitted) return
    skipEmit = true
    local.value = normalizeContent(clone(v))
  },
)

watch(
  local,
  (v) => {
    if (skipEmit) {
      skipEmit = false
      return
    }
    const out = clone(v)
    lastEmitted = out
    emit('update:modelValue', out)
  },
  { deep: true },
)

const uid = useId()
const ids = {
  eventNames: `ev-names-${uid}`,
  bank: `bank-${uid}`,
  ewallet: `ewallet-${uid}`,
  platform: `platform-${uid}`,
}

// ---------- Navigasi section ----------
interface SectionDef {
  id: string
  title: string
  icon: IconName
}

const sections = computed<SectionDef[]>(() => {
  const groom: SectionDef = { id: 'mempelai-pria', title: 'Mempelai Pria', icon: 'user' }
  const bride: SectionDef = { id: 'mempelai-wanita', title: 'Mempelai Wanita', icon: 'heart' }
  const couple = local.value.couple_order === 'bride_first' ? [bride, groom] : [groom, bride]
  return [
    { id: 'umum', title: 'Umum', icon: 'sparkles' },
    ...couple,
    { id: 'acara', title: 'Acara', icon: 'calendar' },
    { id: 'salam', title: 'Salam & Kutipan', icon: 'quote' },
    { id: 'love-story', title: 'Love Story', icon: 'book' },
    { id: 'galeri', title: 'Galeri', icon: 'image' },
    { id: 'live', title: 'Live Streaming', icon: 'video' },
    { id: 'amplop', title: 'Amplop Digital', icon: 'gift' },
    { id: 'rsvp', title: 'RSVP & Ucapan', icon: 'mail' },
    { id: 'musik', title: 'Musik', icon: 'music' },
    { id: 'penutup', title: 'Penutup', icon: 'flag' },
    { id: 'whatsapp', title: 'Pesan WhatsApp', icon: 'message' },
  ]
})

const opened = reactive(new Set<string>(['umum']))
const sectionEl = (id: string) => document.getElementById(`${uid}-${id}`)

function toggle(id: string) {
  if (opened.has(id)) opened.delete(id)
  else opened.add(id)
}

function goTo(id: string) {
  opened.add(id)
  nextTick(() => sectionEl(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}

function expandAll(v: boolean) {
  opened.clear()
  if (v) sections.value.forEach((s) => opened.add(s.id))
}

// ---------- Ringkasan per section ----------
const c = computed(() => local.value)
const summary = computed<Record<string, { text?: string; done?: boolean }>>(() => {
  const v = c.value
  const datedEvents = v.events.filter((e) => e.date).length
  return {
    umum: { text: `${v.event_type === 'ngunduh_mantu' ? 'Ngunduh mantu' : 'Pernikahan'} · ${RELIGIONS.find((r) => r.value === v.religion)?.label ?? ''}` },
    'mempelai-pria': { text: v.groom.full_name || 'Belum diisi', done: !!v.groom.full_name },
    'mempelai-wanita': { text: v.bride.full_name || 'Belum diisi', done: !!v.bride.full_name },
    acara: { text: `${v.events.length} acara${datedEvents < v.events.length ? ` · ${v.events.length - datedEvents} belum bertanggal` : ''}`, done: v.events.length > 0 && datedEvents === v.events.length },
    salam: { text: v.opening.greeting || v.quote.text ? 'Teks kustom' : 'Teks bawaan sesuai agama' },
    'love-story': { text: v.love_story.length ? `${v.love_story.length} cerita` : 'Tidak ditampilkan' },
    galeri: { text: v.gallery.photos.length || v.gallery.video_url ? `${v.gallery.photos.length} foto${v.gallery.video_url ? ' + video' : ''}` : 'Tidak ditampilkan' },
    live: { text: v.live_stream.url ? v.live_stream.platform || 'Aktif' : 'Tidak ditampilkan' },
    amplop: { text: v.gift.enabled ? `${v.gift.accounts.length} rekening${v.gift.confirmation_enabled ? ' · konfirmasi hadiah aktif' : ''}` : 'Nonaktif' },
    rsvp: { text: v.rsvp.enabled ? `Aktif · maks. ${v.rsvp.max_pax} orang` : 'Nonaktif' },
    musik: { text: v.music.enabled ? v.music.title || 'Aktif' : 'Nonaktif' },
    penutup: { text: v.closing.text || v.closing.from ? 'Terisi' : 'Teks bawaan' },
    whatsapp: { text: v.share.whatsapp_template ? 'Template kustom' : 'Template bawaan' },
  }
})

// ---------- Opsi ----------
const EVENT_TYPES: { value: EventType; label: string }[] = [
  { value: 'pernikahan', label: 'Pernikahan' },
  { value: 'ngunduh_mantu', label: 'Ngunduh Mantu' },
]
const RELIGION_OPTS = RELIGIONS.map((r) => ({ value: r.value as Religion, label: r.label }))
const ORDER_OPTS: { value: InvitationContent['couple_order']; label: string }[] = [
  { value: 'groom_first', label: 'Mempelai pria dulu' },
  { value: 'bride_first', label: 'Mempelai wanita dulu' },
]
const TZ_OPTS: { value: Timezone; label: string }[] = [
  { value: 'WIB', label: 'WIB' },
  { value: 'WITA', label: 'WITA' },
  { value: 'WIT', label: 'WIT' },
]
const PAX_OPTS = Array.from({ length: 10 }, (_, i) => ({ value: i + 1, label: `${i + 1} orang` }))
const ACCOUNT_TYPES: { value: 'bank' | 'ewallet'; label: string }[] = [
  { value: 'bank', label: 'Bank' },
  { value: 'ewallet', label: 'E-Wallet' },
]
const PLATFORMS = ['YouTube', 'Instagram Live', 'Zoom', 'Google Meet', 'TikTok Live', 'Facebook Live']

// ---------- Helper list ----------
function move<T>(arr: T[], i: number, dir: -1 | 1) {
  const j = i + dir
  if (j < 0 || j >= arr.length) return
  ;[arr[i], arr[j]] = [arr[j]!, arr[i]!]
}

function addEvent() {
  const names = local.value.events.map((e) => e.name)
  const suggestion = EVENT_NAME_SUGGESTIONS.find((n) => !names.includes(n)) ?? ''
  const ev = newEvent(local.value.events.length === 0 ? (local.value.event_type === 'ngunduh_mantu' ? 'Ngunduh Mantu' : 'Akad Nikah') : suggestion)
  const prev = local.value.events[local.value.events.length - 1]
  if (prev) {
    ev.date = prev.date
    ev.timezone = prev.timezone
    ev.venue = prev.venue
    ev.address = prev.address
    ev.maps_url = prev.maps_url
  }
  local.value.events.push(ev)
}

function addStory() {
  local.value.love_story.push({ date: '', title: '', text: '', photo: '' })
}

function addAccount() {
  local.value.gift.accounts.push({ type: 'bank', provider: '', number: '', holder: '' })
}

function isYoutube(url: string) {
  return !url || /^(https?:\/\/)?(www\.|m\.)?(youtube\.com|youtu\.be)\//i.test(url)
}

function isUrl(url: string) {
  return !url || /^https?:\/\/\S+$/i.test(url)
}

// ---------- WhatsApp ----------
const waInput = ref<InstanceType<typeof UiTextarea> | null>(null)
const defaultTpl = computed(() => defaultWhatsappTemplate(local.value))

function insertToken(token: string) {
  const el = waInput.value?.el
  const cur = local.value.share.whatsapp_template || defaultTpl.value
  if (!el || !local.value.share.whatsapp_template) {
    local.value.share.whatsapp_template = cur + (cur.endsWith(' ') || cur.endsWith('\n') || !cur ? '' : ' ') + token
    return
  }
  const start = el.selectionStart ?? cur.length
  const end = el.selectionEnd ?? cur.length
  local.value.share.whatsapp_template = cur.slice(0, start) + token + cur.slice(end)
  nextTick(() => {
    el.focus()
    el.setSelectionRange(start + token.length, start + token.length)
  })
}

function escapeHtml(s: string) {
  return s.replace(/[&<>"']/g, (ch) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[ch]!)
}

const waPreview = computed(() => {
  const text = fillShareTemplate(local.value.share.whatsapp_template || defaultTpl.value, 'Bapak Andi Wijaya', 'https://nama-anda.undangan.id/K7M2QX')
  return escapeHtml(text)
    .replace(/\*([^*\n]+)\*/g, '<strong>$1</strong>')
    .replace(/(^|\s)_([^_\n]+)_(?=\s|$)/g, '$1<em>$2</em>')
})
</script>

<template>
  <div class="space-y-3" :data-invitation="invitationId">
    <!-- Navigasi section -->
    <nav
      class="sticky top-[var(--editor-nav-top,0px)] z-20 -mx-1 flex items-center gap-1.5 overflow-x-auto bg-[var(--editor-nav-bg,var(--color-surface-2))] px-1 py-2 backdrop-blur [scrollbar-width:none]"
      aria-label="Bagian formulir"
    >
      <button
        v-for="s in sections"
        :key="s.id"
        type="button"
        class="inline-flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition"
        :class="opened.has(s.id) ? 'border-accent/40 bg-accent/10 text-accent' : 'border-line bg-surface text-muted hover:text-ink'"
        @click="goTo(s.id)"
      >
        <UiIcon :name="s.icon" :size="13" />
        {{ s.title }}
      </button>
      <span class="mx-1 h-5 w-px shrink-0 bg-line" />
      <button type="button" class="shrink-0 px-2 py-1.5 text-xs text-muted hover:text-ink" @click="expandAll(opened.size < sections.length)">
        {{ opened.size < sections.length ? 'Buka semua' : 'Tutup semua' }}
      </button>
    </nav>

    <datalist :id="ids.eventNames">
      <option v-for="n in EVENT_NAME_SUGGESTIONS" :key="n" :value="n" />
    </datalist>
    <datalist :id="ids.bank">
      <option v-for="n in BANK_SUGGESTIONS" :key="n" :value="n" />
    </datalist>
    <datalist :id="ids.ewallet">
      <option v-for="n in EWALLET_SUGGESTIONS" :key="n" :value="n" />
    </datalist>
    <datalist :id="ids.platform">
      <option v-for="n in PLATFORMS" :key="n" :value="n" />
    </datalist>

    <template v-for="s in sections" :key="s.id">
      <EditorSection
        :id="`${uid}-${s.id}`"
        :title="s.title"
        :icon="s.icon"
        :open="opened.has(s.id)"
        :summary="summary[s.id]?.text"
        :done="summary[s.id]?.done"
        @toggle="toggle(s.id)"
      >
        <!-- UMUM -->
        <div v-if="s.id === 'umum'" class="space-y-4">
          <div class="grid gap-4 sm:grid-cols-3">
            <UiField label="Jenis acara">
              <UiSelect v-model="local.event_type" :options="EVENT_TYPES" />
            </UiField>
            <UiField label="Agama" hint="Menentukan salam & kutipan bawaan">
              <UiSelect v-model="local.religion" :options="RELIGION_OPTS" />
            </UiField>
            <UiField label="Urutan mempelai">
              <UiSelect v-model="local.couple_order" :options="ORDER_OPTS" />
            </UiField>
          </div>
          <UiField label="Judul sampul" hint="Kosongkan untuk judul bawaan tema">
            <UiInput v-model="local.cover.title" placeholder="The Wedding of" />
          </UiField>
          <div class="grid gap-4 sm:grid-cols-2">
            <UiField label="Foto sampul" hint="Foto utama di halaman pembuka">
              <ImageUpload v-model="local.cover.photo" aspect="3 / 4" />
            </UiField>
            <UiField label="Latar sampul" hint="Opsional">
              <ImageUpload v-model="local.cover.background" aspect="9 / 16" />
            </UiField>
          </div>
        </div>

        <!-- MEMPELAI -->
        <PersonFields v-else-if="s.id === 'mempelai-pria'" :person="local.groom" gender="pria" />
        <PersonFields v-else-if="s.id === 'mempelai-wanita'" :person="local.bride" gender="wanita" />

        <!-- ACARA -->
        <div v-else-if="s.id === 'acara'" class="space-y-3">
          <p v-if="!local.events.length" class="rounded-xl bg-warning/10 px-3 py-2 text-sm text-ink">
            Minimal 1 acara dengan tanggal diperlukan agar undangan bisa dipublikasikan.
          </p>
          <div v-for="(ev, i) in local.events" :key="ev.id" class="rounded-xl border border-line bg-surface-2/50 p-3 sm:p-4">
            <div class="mb-3 flex items-center gap-2">
              <span class="flex size-6 items-center justify-center rounded-full bg-accent text-xs font-semibold text-accent-ink">{{ i + 1 }}</span>
              <span class="min-w-0 flex-1 truncate text-sm font-semibold text-ink">{{ ev.name || 'Acara tanpa nama' }}</span>
              <button type="button" class="rounded-lg p-1.5 text-muted hover:bg-surface hover:text-ink disabled:opacity-30" :disabled="i === 0" aria-label="Naikkan" @click="move(local.events, i, -1)">
                <UiIcon name="arrow-up" :size="15" />
              </button>
              <button type="button" class="rounded-lg p-1.5 text-muted hover:bg-surface hover:text-ink disabled:opacity-30" :disabled="i === local.events.length - 1" aria-label="Turunkan" @click="move(local.events, i, 1)">
                <UiIcon name="arrow-down" :size="15" />
              </button>
              <button type="button" class="rounded-lg p-1.5 text-muted hover:bg-danger/10 hover:text-danger" aria-label="Hapus acara" @click="local.events.splice(i, 1)">
                <UiIcon name="trash" :size="15" />
              </button>
            </div>
            <div class="grid gap-3 sm:grid-cols-6">
              <UiField label="Nama acara" class="sm:col-span-3">
                <UiInput v-model="ev.name" :list="ids.eventNames" placeholder="Akad Nikah" />
              </UiField>
              <UiField label="Tanggal" class="sm:col-span-3">
                <UiInput v-model="ev.date" type="date" />
              </UiField>
              <UiField label="Jam mulai" class="sm:col-span-2">
                <UiInput v-model="ev.start_time" type="time" />
              </UiField>
              <UiField label="Jam selesai" hint="Kosongkan = “Selesai”" class="sm:col-span-2">
                <div class="flex gap-1">
                  <UiInput v-model="ev.end_time" type="time" />
                  <button v-if="ev.end_time" type="button" class="shrink-0 rounded-lg px-2 text-muted hover:text-ink" aria-label="Kosongkan jam selesai" @click="ev.end_time = ''">
                    <UiIcon name="x" :size="14" />
                  </button>
                </div>
              </UiField>
              <UiField label="Zona waktu" class="sm:col-span-2">
                <UiSelect v-model="ev.timezone" :options="TZ_OPTS" />
              </UiField>
              <UiField label="Nama tempat" class="sm:col-span-6">
                <UiInput v-model="ev.venue" placeholder="mis. Gedung Serbaguna Melati" />
              </UiField>
              <UiField label="Alamat" class="sm:col-span-6">
                <UiTextarea v-model="ev.address" :rows="2" placeholder="Jl. …, Kel. …, Kec. …, Kota …" />
              </UiField>
              <UiField label="Link Google Maps" :error="isUrl(ev.maps_url) ? null : 'Link harus diawali https://'" class="sm:col-span-6">
                <UiInput v-model="ev.maps_url" type="url" placeholder="https://maps.app.goo.gl/…" />
              </UiField>
              <UiField label="Catatan" hint="Opsional, mis. “Khusus keluarga”" class="sm:col-span-6">
                <UiInput v-model="ev.note" />
              </UiField>
            </div>
          </div>
          <UiButton variant="secondary" block @click="addEvent"><UiIcon name="plus" :size="16" /> Tambah acara</UiButton>
        </div>

        <!-- SALAM & KUTIPAN -->
        <div v-else-if="s.id === 'salam'" class="space-y-4">
          <p class="flex gap-2 rounded-xl bg-surface-2 px-3 py-2 text-xs text-muted">
            <UiIcon name="info" :size="15" class="mt-px" />
            Kosongkan untuk memakai teks bawaan sesuai agama yang dipilih.
          </p>
          <UiField label="Salam pembuka">
            <UiInput v-model="local.opening.greeting" placeholder="Assalamu'alaikum Warahmatullahi Wabarakatuh" />
          </UiField>
          <UiField label="Paragraf pembuka">
            <UiTextarea v-model="local.opening.text" :rows="3" placeholder="Dengan memohon rahmat dan ridho Allah SWT, kami bermaksud menyelenggarakan …" />
          </UiField>
          <UiField label="Kutipan / ayat">
            <UiTextarea v-model="local.quote.text" :rows="3" />
          </UiField>
          <UiField label="Sumber kutipan">
            <UiInput v-model="local.quote.source" placeholder="mis. QS. Ar-Rum: 21" />
          </UiField>
        </div>

        <!-- LOVE STORY -->
        <div v-else-if="s.id === 'love-story'" class="space-y-3">
          <p v-if="!local.love_story.length" class="text-sm text-muted">Belum ada cerita. Bagian ini tidak ditampilkan jika kosong.</p>
          <div v-for="(st, i) in local.love_story" :key="i" class="rounded-xl border border-line bg-surface-2/50 p-3 sm:p-4">
            <div class="mb-3 flex items-center gap-2">
              <span class="min-w-0 flex-1 truncate text-sm font-semibold text-ink">{{ st.title || `Cerita ${i + 1}` }}</span>
              <button type="button" class="rounded-lg p-1.5 text-muted hover:bg-surface hover:text-ink disabled:opacity-30" :disabled="i === 0" aria-label="Naikkan" @click="move(local.love_story, i, -1)">
                <UiIcon name="arrow-up" :size="15" />
              </button>
              <button type="button" class="rounded-lg p-1.5 text-muted hover:bg-surface hover:text-ink disabled:opacity-30" :disabled="i === local.love_story.length - 1" aria-label="Turunkan" @click="move(local.love_story, i, 1)">
                <UiIcon name="arrow-down" :size="15" />
              </button>
              <button type="button" class="rounded-lg p-1.5 text-muted hover:bg-danger/10 hover:text-danger" aria-label="Hapus cerita" @click="local.love_story.splice(i, 1)">
                <UiIcon name="trash" :size="15" />
              </button>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <UiField label="Waktu" hint="Bebas, mis. “Maret 2019”">
                <UiInput v-model="st.date" placeholder="Maret 2019" />
              </UiField>
              <UiField label="Judul">
                <UiInput v-model="st.title" placeholder="Pertama bertemu" />
              </UiField>
              <UiField label="Cerita" class="sm:col-span-2">
                <UiTextarea v-model="st.text" :rows="3" />
              </UiField>
              <UiField label="Foto" hint="Opsional" class="sm:col-span-2">
                <ImageUpload v-model="st.photo" aspect="4 / 3" />
              </UiField>
            </div>
          </div>
          <UiButton variant="secondary" block @click="addStory"><UiIcon name="plus" :size="16" /> Tambah cerita</UiButton>
        </div>

        <!-- GALERI -->
        <div v-else-if="s.id === 'galeri'" class="space-y-4">
          <UiField label="Foto galeri" hint="Maksimal 20 foto">
            <ImageUpload v-model="local.gallery.photos" multiple :max="20" aspect="1 / 1" />
          </UiField>
          <UiField label="Video YouTube" :error="isYoutube(local.gallery.video_url) ? null : 'Masukkan link YouTube yang valid'" hint="Opsional, mis. video prewedding">
            <UiInput v-model="local.gallery.video_url" type="url" placeholder="https://www.youtube.com/watch?v=…" />
          </UiField>
        </div>

        <!-- LIVE STREAMING -->
        <div v-else-if="s.id === 'live'" class="space-y-4">
          <p class="text-sm text-muted">Kosongkan link untuk menyembunyikan bagian ini.</p>
          <UiField label="Link siaran" :error="isUrl(local.live_stream.url) ? null : 'Link harus diawali https://'">
            <UiInput v-model="local.live_stream.url" type="url" placeholder="https://youtube.com/live/…" />
          </UiField>
          <div class="grid gap-4 sm:grid-cols-2">
            <UiField label="Platform">
              <UiInput v-model="local.live_stream.platform" :list="ids.platform" placeholder="YouTube" />
            </UiField>
            <UiField label="Catatan">
              <UiInput v-model="local.live_stream.note" placeholder="mis. Mulai pukul 08.00 WIB" />
            </UiField>
          </div>
        </div>

        <!-- AMPLOP DIGITAL -->
        <div v-else-if="s.id === 'amplop'" class="space-y-4">
          <UiSwitch v-model="local.gift.enabled" label="Tampilkan amplop digital" description="Tamu dapat menyalin nomor rekening / e-wallet." />
          <template v-if="local.gift.enabled">
            <UiField label="Teks pengantar">
              <UiTextarea v-model="local.gift.text" :rows="2" placeholder="Doa restu Anda merupakan karunia yang sangat berarti bagi kami. Namun jika memberi adalah ungkapan tanda kasih, Anda dapat memberi kado secara cashless." />
            </UiField>
            <div class="space-y-3">
              <p class="text-sm font-medium text-ink">Rekening & e-wallet</p>
              <div v-for="(acc, i) in local.gift.accounts" :key="i" class="grid gap-3 rounded-xl border border-line bg-surface-2/50 p-3 sm:grid-cols-12">
                <UiField label="Jenis" class="sm:col-span-3">
                  <UiSelect v-model="acc.type" :options="ACCOUNT_TYPES" />
                </UiField>
                <UiField :label="acc.type === 'bank' ? 'Bank' : 'E-Wallet'" class="sm:col-span-4">
                  <UiInput v-model="acc.provider" :list="acc.type === 'bank' ? ids.bank : ids.ewallet" :placeholder="acc.type === 'bank' ? 'BCA' : 'DANA'" />
                </UiField>
                <UiField label="Nomor" class="sm:col-span-5">
                  <UiInput v-model="acc.number" inputmode="numeric" placeholder="1234567890" />
                </UiField>
                <UiField label="Atas nama" class="sm:col-span-10">
                  <UiInput v-model="acc.holder" />
                </UiField>
                <div class="flex items-end sm:col-span-2">
                  <UiButton variant="danger" size="md" block @click="local.gift.accounts.splice(i, 1)">
                    <UiIcon name="trash" :size="15" /><span class="sm:hidden">Hapus</span>
                  </UiButton>
                </div>
              </div>
              <UiButton variant="secondary" block @click="addAccount"><UiIcon name="plus" :size="16" /> Tambah rekening</UiButton>
            </div>
            <div class="rounded-xl border border-line p-3 sm:p-4">
              <UiSwitch
                v-model="local.gift.confirmation_enabled"
                label="Izinkan tamu mengirim bukti transfer/kado"
                description="Tamu dapat mengunggah foto bukti transfer atau kado. Bukti bersifat privat dan bisa dicek di menu Hadiah."
              />
            </div>
            <div class="space-y-3 rounded-xl border border-line p-3 sm:p-4">
              <div>
                <p class="text-sm font-medium text-ink">Alamat kirim kado</p>
                <p class="text-xs text-muted">Opsional — kosongkan alamat untuk menyembunyikan.</p>
              </div>
              <div class="grid gap-3 sm:grid-cols-2">
                <UiField label="Penerima">
                  <UiInput v-model="local.gift.address.recipient" />
                </UiField>
                <UiField label="No. HP penerima">
                  <UiInput v-model="local.gift.address.phone" type="tel" placeholder="08…" />
                </UiField>
                <UiField label="Alamat lengkap" class="sm:col-span-2">
                  <UiTextarea v-model="local.gift.address.address" :rows="2" />
                </UiField>
              </div>
            </div>
          </template>
        </div>

        <!-- RSVP -->
        <div v-else-if="s.id === 'rsvp'" class="space-y-4">
          <UiSwitch v-model="local.rsvp.enabled" label="Aktifkan RSVP & ucapan" description="Tamu mengisi kehadiran, jumlah orang, dan ucapan." />
          <template v-if="local.rsvp.enabled">
            <UiField label="Maksimal jumlah tamu per RSVP" hint="Pilihan jumlah orang di formulir (1–10)" class="sm:max-w-xs">
              <UiSelect v-model="local.rsvp.max_pax" :options="PAX_OPTS" />
            </UiField>
            <UiSwitch v-model="local.rsvp.show_wishes" label="Tampilkan daftar ucapan" description="Ucapan dari tamu tampil di halaman undangan." />
          </template>
        </div>

        <!-- MUSIK -->
        <MusicPicker v-else-if="s.id === 'musik'" v-model="local.music" />

        <!-- PENUTUP -->
        <div v-else-if="s.id === 'penutup'" class="space-y-4">
          <UiField label="Teks penutup" hint="Kosongkan untuk teks bawaan">
            <UiTextarea v-model="local.closing.text" :rows="3" placeholder="Merupakan suatu kehormatan dan kebahagiaan bagi kami apabila Bapak/Ibu/Saudara/i berkenan hadir …" />
          </UiField>
          <div class="grid gap-4 sm:grid-cols-2">
            <UiField label="Salam penutup" hint="Kosongkan untuk salam bawaan sesuai agama">
              <UiInput v-model="local.closing.sign_off" placeholder="Wassalamu'alaikum Warahmatullahi Wabarakatuh" />
            </UiField>
            <UiField label="Dari">
              <UiInput v-model="local.closing.from" placeholder="Kami yang berbahagia, Kedua Mempelai & Keluarga Besar" />
            </UiField>
          </div>
          <div class="space-y-2">
            <p class="text-sm font-medium text-ink">Turut mengundang <span class="font-normal text-muted">(opsional)</span></p>
            <div v-for="(_, i) in local.closing.family" :key="i" class="flex gap-2">
              <UiInput v-model="local.closing.family[i]" placeholder="mis. Keluarga Besar Bapak Suyadi" />
              <button type="button" class="shrink-0 rounded-xl border border-line px-3 text-muted hover:bg-danger/10 hover:text-danger" aria-label="Hapus" @click="local.closing.family.splice(i, 1)">
                <UiIcon name="trash" :size="15" />
              </button>
            </div>
            <UiButton variant="secondary" size="sm" @click="local.closing.family.push('')"><UiIcon name="plus" :size="14" /> Tambah nama</UiButton>
          </div>
        </div>

        <!-- WHATSAPP -->
        <div v-else-if="s.id === 'whatsapp'" class="space-y-4">
          <p class="text-sm text-muted">Pesan ini dipakai saat membagikan undangan ke tamu lewat WhatsApp.</p>
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-xs text-muted">Sisipkan:</span>
            <button type="button" class="rounded-full border border-accent/30 bg-accent/10 px-2.5 py-1 font-mono text-xs text-accent hover:bg-accent/15" @click="insertToken('{nama}')">{nama}</button>
            <button type="button" class="rounded-full border border-accent/30 bg-accent/10 px-2.5 py-1 font-mono text-xs text-accent hover:bg-accent/15" @click="insertToken('{link}')">{link}</button>
            <span class="flex-1" />
            <button v-if="local.share.whatsapp_template" type="button" class="text-xs text-muted underline hover:text-ink" @click="local.share.whatsapp_template = ''">
              Kembalikan ke bawaan
            </button>
            <button v-else type="button" class="text-xs text-muted underline hover:text-ink" @click="local.share.whatsapp_template = defaultTpl">
              Edit dari template bawaan
            </button>
          </div>
          <UiTextarea ref="waInput" v-model="local.share.whatsapp_template" :rows="8" :placeholder="defaultTpl" />
          <p class="text-xs text-muted">
            <code class="font-mono">{nama}</code> diganti nama tamu, <code class="font-mono">{link}</code> diganti link undangan personal. Gunakan *teks* untuk huruf tebal.
          </p>
          <div>
            <p class="mb-2 text-xs font-medium text-muted uppercase">Pratinjau</p>
            <div class="rounded-2xl bg-surface-2 p-3">
              <div class="max-w-md rounded-xl rounded-tl-sm border border-line bg-surface px-3 py-2 text-sm leading-relaxed break-words whitespace-pre-wrap text-ink shadow-sm" v-html="waPreview" />
            </div>
          </div>
        </div>
      </EditorSection>
    </template>
  </div>
</template>
