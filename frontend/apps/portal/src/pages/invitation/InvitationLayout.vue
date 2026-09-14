<script setup lang="ts">
import { ApiError, api, errorMessage, normalizeContent, type Invitation } from '@undangan/shared'
import UiBadge from '@undangan/shared/components/ui/UiBadge.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiIcon, { type IconName } from '@undangan/shared/components/ui/UiIcon.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { INVITATION_CTX } from '../../lib/invitation'
import { INV_STATUS } from '../../lib/labels'

const props = defineProps<{ id: string }>()
const route = useRoute()

const invitation = ref<Invitation | null>(null)
const loading = ref(true)
const error = ref('')
const notFound = ref(false)

function setInvitation(inv: Invitation) {
  invitation.value = { ...inv, content: normalizeContent(inv.content) }
}

async function fetchInvitation() {
  const inv = await api.get<Invitation>(`/invitations/${props.id}`)
  setInvitation(inv)
}

async function load() {
  loading.value = true
  error.value = ''
  notFound.value = false
  try {
    await fetchInvitation()
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound.value = true
    else error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

async function reload() {
  try {
    await fetchInvitation()
  } catch {
    /* abaikan; data lama tetap dipakai */
  }
}

watch(() => props.id, load, { immediate: true })

provide(INVITATION_CTX, { invitation, setInvitation, reload })

// Tinggi header sticky → dipakai child (nav editor & pratinjau) sebagai offset sticky.
const headerEl = ref<HTMLElement | null>(null)
const headerBottom = ref(0)
let ro: ResizeObserver | null = null
function measure() {
  const el = headerEl.value
  if (!el) return
  headerBottom.value = Math.round(el.offsetHeight + (parseFloat(getComputedStyle(el).top) || 0))
}
onMounted(() => {
  measure()
  ro = new ResizeObserver(measure)
  if (headerEl.value) ro.observe(headerEl.value)
  window.addEventListener('resize', measure)
})
onBeforeUnmount(() => {
  ro?.disconnect()
  window.removeEventListener('resize', measure)
})

const tabs: { name: string; path: string; label: string; icon: IconName }[] = [
  { name: 'inv-data', path: 'data', label: 'Isi Data', icon: 'pencil' },
  { name: 'inv-theme', path: 'tema', label: 'Tema', icon: 'palette' },
  { name: 'inv-guests', path: 'tamu', label: 'Tamu', icon: 'users' },
  { name: 'inv-attendance', path: 'kehadiran', label: 'Kehadiran', icon: 'user-check' },
  { name: 'inv-wishes', path: 'ucapan', label: 'Ucapan', icon: 'heart' },
  { name: 'inv-gifts', path: 'hadiah', label: 'Hadiah', icon: 'gift' },
  { name: 'inv-share', path: 'bagikan', label: 'Bagikan', icon: 'share' },
]

const title = computed(() => {
  const inv = invitation.value
  if (!inv) return ''
  const c = inv.content
  const g = c.groom.nickname || c.groom.full_name
  const b = c.bride.nickname || c.bride.full_name
  if (g && b) return c.couple_order === 'bride_first' ? `${b} & ${g}` : `${g} & ${b}`
  return inv.title || 'Undangan baru'
})
</script>

<template>
  <div :style="{ '--editor-nav-top': `${headerBottom}px` }">
    <!-- Header undangan -->
    <div ref="headerEl" class="sticky top-0 z-30 border-b border-line/80 bg-page/90 backdrop-blur md:top-16">
      <div class="mx-auto max-w-7xl px-4 pt-[max(0.75rem,env(safe-area-inset-top))] md:px-6 md:pt-4">
        <div class="flex items-center gap-3">
          <RouterLink to="/undangan" class="-ml-2 rounded-full p-2 text-muted hover:bg-surface-2 hover:text-ink" aria-label="Kembali ke daftar undangan">
            <UiIcon name="arrow-left" :size="20" />
          </RouterLink>
          <div class="min-w-0 flex-1">
            <h1 class="heading truncate text-2xl leading-tight sm:text-3xl">{{ loading ? 'Memuat…' : title }}</h1>
            <p v-if="invitation" class="flex items-center gap-2 truncate text-xs text-muted">
              <UiBadge :tone="INV_STATUS[invitation.status]?.tone ?? 'neutral'" class="py-0!">{{ INV_STATUS[invitation.status]?.label }}</UiBadge>
              <span v-if="invitation.url" class="truncate">{{ invitation.url.replace(/^https?:\/\//, '') }}</span>
              <span v-else>Subdomain belum diatur</span>
            </p>
          </div>
          <UiButton
            v-if="invitation?.url && invitation.status === 'published'"
            variant="secondary"
            size="sm"
            :href="invitation.url"
            target="_blank"
            rel="noopener"
            class="hidden sm:inline-flex"
          >
            <UiIcon name="external" :size="14" /> Buka
          </UiButton>
        </div>
        <nav class="-mx-4 mt-2 flex gap-1 overflow-x-auto px-4 [scrollbar-width:none] md:mx-0 md:px-0" aria-label="Menu undangan">
          <RouterLink
            v-for="t in tabs"
            :key="t.name"
            :to="`/undangan/${id}/${t.path}`"
            class="flex shrink-0 items-center gap-1.5 border-b-2 px-3 py-2.5 text-sm font-medium transition"
            :class="route.name === t.name ? 'border-accent text-accent' : 'border-transparent text-muted hover:text-ink'"
          >
            <UiIcon :name="t.icon" :size="15" />
            {{ t.label }}
          </RouterLink>
        </nav>
      </div>
    </div>

    <div v-if="loading" class="flex items-center justify-center gap-2 py-24 text-sm text-muted"><UiSpinner /> Memuat undangan…</div>
    <div v-else-if="notFound" class="mx-auto max-w-lg px-4 py-16 text-center">
      <p class="heading text-3xl">Undangan tidak ditemukan</p>
      <p class="mt-1 text-sm text-muted">Undangan mungkin sudah dihapus atau bukan milik akun Anda.</p>
      <UiButton class="mt-4" href="/undangan" @click.prevent="$router.push('/undangan')">Kembali ke daftar</UiButton>
    </div>
    <div v-else-if="error" class="mx-auto max-w-lg px-4 py-16 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <UiButton variant="secondary" size="sm" class="mt-3" @click="load"><UiIcon name="refresh" :size="14" /> Coba lagi</UiButton>
    </div>
    <div v-else-if="invitation" class="mx-auto max-w-7xl px-4 pt-5 md:px-6 md:pt-6">
      <RouterView :key="id" />
    </div>
  </div>
</template>
