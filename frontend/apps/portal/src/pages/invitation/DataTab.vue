<script setup lang="ts">
import { api, confirm, toast, useMediaQuery, type Invitation, type InvitationContent } from '@undangan/shared'
import InvitationEditor from '@undangan/shared/components/InvitationEditor.vue'
import PreviewFrame from '@undangan/shared/components/PreviewFrame.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiDialog from '@undangan/shared/components/ui/UiDialog.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { useInvitationCtx } from '../../lib/invitation'

const ctx = useInvitationCtx()
const inv = computed(() => ctx.invitation.value as Invitation)

const content = ref<InvitationContent>(JSON.parse(JSON.stringify(inv.value.content)))
const savedJson = ref(JSON.stringify(content.value))
const currentJson = computed(() => JSON.stringify(content.value))
const dirty = computed(() => currentJson.value !== savedJson.value)
const saving = ref(false)
const lastSavedAt = ref<Date | null>(null)

const reloadKey = ref(0)
const device = ref<'mobile' | 'desktop'>('mobile')
const previewSrc = computed(() => api.url(`/invitations/${inv.value.id}/preview`))
const isDesktop = useMediaQuery('(min-width: 1024px)')
const isTablet = useMediaQuery('(min-width: 640px)')
const previewOpen = ref(false)

async function save() {
  if (saving.value || !dirty.value) return
  saving.value = true
  const snapshot = currentJson.value
  try {
    const updated = await api.patch<Invitation>(`/invitations/${inv.value.id}`, { content: JSON.parse(snapshot) })
    ctx.setInvitation(updated)
    savedJson.value = snapshot
    lastSavedAt.value = new Date()
    reloadKey.value++
    toast.success('Perubahan disimpan')
  } catch (e) {
    toast.error(e)
  } finally {
    saving.value = false
  }
}

function discard() {
  content.value = JSON.parse(savedJson.value)
}

async function askDiscard() {
  const ok = await confirm({ title: 'Batalkan perubahan?', message: 'Semua perubahan yang belum disimpan akan hilang.', confirmText: 'Batalkan perubahan', danger: true })
  if (ok) discard()
}

function onKey(e: KeyboardEvent) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    save()
  }
}

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (dirty.value) {
    e.preventDefault()
    e.returnValue = ''
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKey)
  window.addEventListener('beforeunload', onBeforeUnload)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('beforeunload', onBeforeUnload)
})

onBeforeRouteLeave(async () => {
  if (!dirty.value) return true
  return confirm({
    title: 'Perubahan belum disimpan',
    message: 'Anda punya perubahan yang belum disimpan. Tinggalkan halaman ini dan buang perubahan?',
    confirmText: 'Tinggalkan',
    danger: true,
  })
})

function openPreview() {
  if (dirty.value) toast.info('Pratinjau menampilkan data yang sudah disimpan.')
  previewOpen.value = true
}

const timeFmt = new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit' })
</script>

<template>
  <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_400px] xl:grid-cols-[minmax(0,1fr)_440px]">
    <div class="min-w-0">
      <div v-if="!inv.theme" class="mb-3 flex items-center gap-3 rounded-2xl border border-accent/25 bg-accent/8 px-4 py-3 text-sm">
        <UiIcon name="palette" :size="18" class="shrink-0 text-accent" />
        <p class="flex-1 text-ink">Belum memilih tema. Isi data dulu, lalu pilih tema yang paling cocok.</p>
        <RouterLink :to="`/undangan/${inv.id}/tema`" class="shrink-0 font-medium text-accent hover:underline">Pilih tema</RouterLink>
      </div>

      <InvitationEditor v-model="content" :invitation-id="inv.id" />

      <!-- Bar simpan -->
      <div class="sticky bottom-0 z-20 -mx-4 mt-4 px-4 pt-2 pb-[max(0.75rem,env(safe-area-inset-bottom))] md:mx-0 md:px-0">
        <div
          class="flex items-center gap-2 rounded-2xl border px-3 py-2.5 shadow-lift transition sm:gap-3 sm:px-4"
          :class="dirty ? 'border-accent/30 bg-surface' : 'border-line bg-surface/95'"
        >
          <div class="flex min-w-0 flex-1 items-center gap-2 text-sm">
            <span class="size-2 shrink-0 rounded-full" :class="dirty ? 'animate-pulse bg-warning' : 'bg-success'" />
            <span v-if="dirty" class="truncate font-medium text-ink">Perubahan belum disimpan</span>
            <span v-else class="truncate text-muted">
              {{ lastSavedAt ? `Tersimpan pukul ${timeFmt.format(lastSavedAt)}` : 'Semua perubahan tersimpan' }}
            </span>
          </div>
          <UiButton variant="ghost" size="sm" class="lg:hidden" @click="openPreview"><UiIcon name="eye" :size="15" /> <span class="max-sm:hidden">Pratinjau</span></UiButton>
          <UiButton v-if="dirty" variant="ghost" size="sm" class="max-sm:hidden" :disabled="saving" @click="askDiscard">Batalkan</UiButton>
          <UiButton :disabled="!dirty" :loading="saving" @click="save">
            <UiIcon v-if="!saving" name="save" :size="16" /> Simpan
            <kbd class="ml-1 hidden rounded border border-accent-ink/30 px-1 text-[10px] font-normal opacity-80 xl:inline">Ctrl S</kbd>
          </UiButton>
        </div>
      </div>
    </div>

    <!-- Pratinjau desktop -->
    <aside v-if="isDesktop" class="hidden lg:block">
      <div class="sticky top-[calc(var(--editor-nav-top,0px)+1rem)] flex h-[calc(100dvh-var(--editor-nav-top,0px)-2rem)] min-h-[520px] flex-col">
        <div class="mb-2 flex items-center justify-between gap-2">
          <p class="text-sm font-medium text-ink">Pratinjau</p>
          <div class="flex items-center gap-1">
            <div class="flex items-center gap-0.5 rounded-lg bg-surface-2 p-0.5">
              <button
                type="button"
                class="rounded-md p-1.5 transition"
                :class="device === 'mobile' ? 'bg-surface text-ink shadow-sm' : 'text-muted'"
                aria-label="Tampilan ponsel"
                @click="device = 'mobile'"
              >
                <UiIcon name="smartphone" :size="15" />
              </button>
              <button
                type="button"
                class="rounded-md p-1.5 transition"
                :class="device === 'desktop' ? 'bg-surface text-ink shadow-sm' : 'text-muted'"
                aria-label="Tampilan desktop"
                @click="device = 'desktop'"
              >
                <UiIcon name="monitor" :size="15" />
              </button>
            </div>
            <button type="button" class="rounded-lg p-2 text-muted hover:bg-surface-2 hover:text-ink" aria-label="Muat ulang pratinjau" @click="reloadKey++">
              <UiIcon name="refresh" :size="15" />
            </button>
            <a :href="previewSrc" target="_blank" rel="noopener" class="rounded-lg p-2 text-muted hover:bg-surface-2 hover:text-ink" aria-label="Buka pratinjau di tab baru">
              <UiIcon name="external" :size="15" />
            </a>
          </div>
        </div>
        <div class="min-h-0 flex-1">
          <PreviewFrame :src="previewSrc" :reload-key="reloadKey" :device="device" />
        </div>
        <p class="mt-2 text-center text-xs text-muted">
          {{ dirty ? 'Simpan untuk melihat perubahan terbaru di pratinjau.' : 'Pratinjau sesuai data tersimpan.' }}
        </p>
      </div>
    </aside>

    <UiDialog v-model:open="previewOpen" title="Pratinjau undangan" size="full" flush>
      <template #actions>
        <button type="button" class="rounded-lg p-1.5 text-muted hover:bg-surface-2 hover:text-ink" aria-label="Muat ulang" @click="reloadKey++">
          <UiIcon name="refresh" :size="18" />
        </button>
      </template>
      <div class="h-full bg-surface-2 sm:p-4">
        <PreviewFrame :src="previewSrc" :reload-key="reloadKey" :device="isTablet ? 'mobile' : 'desktop'" />
      </div>
    </UiDialog>
  </div>
</template>
