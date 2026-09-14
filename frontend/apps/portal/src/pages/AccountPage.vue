<script setup lang="ts">
import { ApiError, api, confirm, formatDate, toast, type Subscription, type User } from '@undangan/shared'
import UiBadge from '@undangan/shared/components/ui/UiBadge.vue'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiField from '@undangan/shared/components/ui/UiField.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiInput from '@undangan/shared/components/ui/UiInput.vue'
import UiSpinner from '@undangan/shared/components/ui/UiSpinner.vue'
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import { auth, logout } from '../lib/auth'
import { SUB_STATUS } from '../lib/labels'

const router = useRouter()

// ---------- Profil ----------
const profile = reactive({ name: '', email: '', phone: '' })
const profileErrors = ref<Record<string, string>>({})
const loadingProfile = ref(true)
const savingProfile = ref(false)

function fill(u: User | null) {
  profile.name = u?.name ?? ''
  profile.email = u?.email ?? ''
  profile.phone = u?.phone ?? ''
}

async function loadProfile() {
  fill(auth.user)
  try {
    const res = await api.get<{ user: User }>('/me/profile')
    auth.user = res.user
    fill(res.user)
  } catch {
    /* pakai data sesi */
  } finally {
    loadingProfile.value = false
  }
}

async function saveProfile() {
  profileErrors.value = {}
  if (!profile.name.trim()) {
    profileErrors.value = { name: 'Nama wajib diisi' }
    return
  }
  savingProfile.value = true
  try {
    const res = await api.patch<{ user: User }>('/me/profile', { name: profile.name.trim(), phone: profile.phone.trim() })
    auth.user = res.user
    fill(res.user)
    toast.success('Profil disimpan')
  } catch (e) {
    if (e instanceof ApiError && Object.keys(e.fields).length) profileErrors.value = e.fields
    else toast.error(e)
  } finally {
    savingProfile.value = false
  }
}

// ---------- Kata sandi ----------
const pw = reactive({ current: '', next: '', confirm: '' })
const pwErrors = ref<Record<string, string>>({})
const savingPw = ref(false)

async function changePassword() {
  const f: Record<string, string> = {}
  if (!pw.current) f.current_password = 'Masukkan kata sandi saat ini'
  if (pw.next.length < 8) f.new_password = 'Minimal 8 karakter'
  if (pw.confirm !== pw.next) f.confirm = 'Konfirmasi tidak sama'
  pwErrors.value = f
  if (Object.keys(f).length) return
  savingPw.value = true
  try {
    await api.post('/me/password', { current_password: pw.current, new_password: pw.next })
    Object.assign(pw, { current: '', next: '', confirm: '' })
    toast.success('Kata sandi berhasil diubah')
  } catch (e) {
    if (e instanceof ApiError && Object.keys(e.fields).length) pwErrors.value = e.fields
    else toast.error(e)
  } finally {
    savingPw.value = false
  }
}

// ---------- Riwayat langganan ----------
const history = ref<Subscription[]>([])
async function loadSubs() {
  try {
    const res = await api.get<{ current: Subscription | null; history: Subscription[] }>('/me/subscription')
    history.value = res.history ?? []
  } catch {
    history.value = []
  }
}

async function doLogout() {
  const ok = await confirm({ title: 'Keluar?', message: 'Anda akan keluar dari portal.', confirmText: 'Keluar' })
  if (!ok) return
  await logout()
  router.replace('/masuk')
}

onMounted(() => {
  loadProfile()
  loadSubs()
})
</script>

<template>
  <div class="mx-auto max-w-3xl px-4 pt-2 md:px-6 md:pt-10">
    <PageHeader title="Akun" subtitle="Kelola profil dan keamanan akun Anda." />

    <div class="space-y-5">
      <section class="card p-5 sm:p-6">
        <div class="mb-5 flex items-center gap-3">
          <span class="flex size-12 items-center justify-center rounded-full bg-accent-soft text-lg font-semibold text-accent">
            {{ (profile.name || '?').charAt(0).toUpperCase() }}
          </span>
          <div class="min-w-0">
            <h2 class="heading truncate text-2xl">{{ profile.name || 'Profil' }}</h2>
            <p class="truncate text-sm text-muted">{{ profile.email }}</p>
          </div>
          <UiSpinner v-if="loadingProfile" class="ml-auto text-muted" />
        </div>
        <form class="space-y-4" @submit.prevent="saveProfile">
          <UiField label="Nama lengkap" :error="profileErrors.name">
            <UiInput v-model="profile.name" autocomplete="name" :invalid="!!profileErrors.name" />
          </UiField>
          <div class="grid gap-4 sm:grid-cols-2">
            <UiField label="Email" hint="Hubungi admin untuk mengganti email" :error="profileErrors.email">
              <UiInput v-model="profile.email" type="email" readonly />
            </UiField>
            <UiField label="No. WhatsApp" :error="profileErrors.phone">
              <UiInput v-model="profile.phone" type="tel" inputmode="tel" autocomplete="tel" placeholder="081234567890" :invalid="!!profileErrors.phone" />
            </UiField>
          </div>
          <div class="flex justify-end">
            <UiButton type="submit" :loading="savingProfile">Simpan profil</UiButton>
          </div>
        </form>
      </section>

      <section class="card p-5 sm:p-6">
        <h2 class="heading mb-1 flex items-center gap-2 text-2xl"><UiIcon name="key" :size="18" class="text-accent" /> Ubah kata sandi</h2>
        <p class="mb-5 text-sm text-muted">Gunakan minimal 8 karakter.</p>
        <form class="space-y-4" @submit.prevent="changePassword">
          <UiField label="Kata sandi saat ini" :error="pwErrors.current_password">
            <UiInput v-model="pw.current" type="password" autocomplete="current-password" :invalid="!!pwErrors.current_password" />
          </UiField>
          <div class="grid gap-4 sm:grid-cols-2">
            <UiField label="Kata sandi baru" :error="pwErrors.new_password">
              <UiInput v-model="pw.next" type="password" autocomplete="new-password" :invalid="!!pwErrors.new_password" />
            </UiField>
            <UiField label="Ulangi kata sandi baru" :error="pwErrors.confirm">
              <UiInput v-model="pw.confirm" type="password" autocomplete="new-password" :invalid="!!pwErrors.confirm" />
            </UiField>
          </div>
          <div class="flex justify-end">
            <UiButton type="submit" :loading="savingPw">Ubah kata sandi</UiButton>
          </div>
        </form>
      </section>

      <section v-if="history.length" class="card p-5 sm:p-6">
        <h2 class="heading mb-4 text-2xl">Riwayat langganan</h2>
        <ul class="divide-y divide-line text-sm">
          <li v-for="s in history" :key="s.id" class="flex items-center justify-between gap-3 py-3">
            <div class="min-w-0">
              <p class="font-medium text-ink">{{ s.plan_name }}</p>
              <p class="text-xs text-muted">{{ formatDate(s.starts_at) }} – {{ formatDate(s.ends_at) }}</p>
            </div>
            <UiBadge :tone="SUB_STATUS[s.status]?.tone ?? 'neutral'">{{ SUB_STATUS[s.status]?.label ?? s.status }}</UiBadge>
          </li>
        </ul>
        <RouterLink to="/pembayaran" class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline">
          Riwayat pembayaran <UiIcon name="chevron-right" :size="14" />
        </RouterLink>
      </section>

      <section class="card flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:p-6">
        <div class="flex-1">
          <p class="font-medium text-ink">Keluar dari akun</p>
          <p class="text-sm text-muted">Anda perlu masuk kembali untuk mengelola undangan.</p>
        </div>
        <UiButton variant="danger" @click="doLogout"><UiIcon name="logout" :size="16" /> Keluar</UiButton>
      </section>
    </div>
  </div>
</template>
