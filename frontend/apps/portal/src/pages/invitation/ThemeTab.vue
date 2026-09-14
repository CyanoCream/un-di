<script setup lang="ts">
import { ApiError, api, errorMessage, toast, type Invitation, type ThemeInfo } from '@undangan/shared'
import ThemePicker from '@undangan/shared/components/ThemePicker.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { computed, onMounted, ref } from 'vue'
import { useInvitationCtx } from '../../lib/invitation'
import { clearPendingTheme, getPendingTheme } from '../../lib/pendingTheme'

const ctx = useInvitationCtx()
const inv = computed(() => ctx.invitation.value as Invitation)

const themes = ref<ThemeInfo[]>([])
const loading = ref(true)
const error = ref('')
const busy = ref(false)

// Tema yang diminati dari landing page — hanya disorot untuk undangan yang belum punya tema.
const pendingTheme = ref(getPendingTheme())
const highlight = computed(() =>
  !inv.value.theme && pendingTheme.value && themes.value.some((t) => t.slug === pendingTheme.value) ? pendingTheme.value : null,
)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<{ items: ThemeInfo[] }>('/themes')
    themes.value = res.items ?? []
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

async function select(slug: string) {
  busy.value = true
  try {
    const updated = await api.put<Invitation>(`/invitations/${inv.value.id}/theme`, { theme: slug })
    ctx.setInvitation(updated)
    clearPendingTheme()
    pendingTheme.value = null
    toast.success(`Tema ${themes.value.find((t) => t.slug === slug)?.name ?? slug} dipilih`)
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) {
      toast.error('Tema sudah terkunci. Hubungi admin untuk mengganti tema.')
      ctx.reload()
    } else {
      toast.error(e)
    }
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="pb-6">
    <div class="mb-5">
      <h2 class="heading text-3xl">Pilih tema</h2>
      <p class="mt-1 text-sm text-muted">Klik “Pratinjau” untuk melihat tema dengan data undangan Anda sebelum memilih.</p>
    </div>
    <div v-if="loading" class="flex items-center justify-center gap-2 py-20 text-sm text-muted"><UiSpinner /> Memuat tema…</div>
    <div v-else-if="error" class="card p-8 text-center">
      <p class="text-sm text-danger">{{ error }}</p>
      <UiButton variant="secondary" size="sm" class="mt-3" @click="load"><UiIcon name="refresh" :size="14" /> Coba lagi</UiButton>
    </div>
    <ThemePicker
      v-else
      :themes="themes"
      :current="inv.theme"
      :locked="inv.theme_locked"
      :can-override-lock="false"
      :invitation-id="inv.id"
      :busy="busy"
      :highlight="highlight"
      @select="select"
    />
  </div>
</template>
