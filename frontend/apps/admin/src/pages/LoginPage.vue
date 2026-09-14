<script setup lang="ts">
import { ApiError } from '@undangan/shared'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import UiSpinner from '../components/UiSpinner.vue'
import { login } from '../lib/auth'
import { errMsg } from '../lib/util'

const route = useRoute()
const router = useRouter()

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')

const notice = computed(() => {
  if (error.value) return error.value
  if (route.query.error === 'role') return 'Akun ini tidak memiliki akses ke Admin Console.'
  if (route.query.error === 'session') return 'Sesi Anda telah berakhir. Silakan masuk kembali.'
  return ''
})

async function submit() {
  if (!email.value || !password.value) {
    error.value = 'Email dan password wajib diisi.'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await login(email.value.trim(), password.value)
    const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/') ? route.query.redirect : '/'
    router.replace(redirect)
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) error.value = 'Email atau password salah.'
    else if (e instanceof ApiError && e.status === 403) error.value = 'Akun ini tidak memiliki akses ke Admin Console.'
    else if (e instanceof ApiError && e.status === 429) error.value = 'Terlalu banyak percobaan. Coba lagi beberapa saat.'
    else error.value = errMsg(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="relative flex min-h-dvh items-center justify-center overflow-hidden px-4 py-10">
    <div
      class="pointer-events-none absolute inset-0 opacity-[0.35]"
      style="
        background-image: linear-gradient(var(--color-line) 1px, transparent 1px), linear-gradient(90deg, var(--color-line) 1px, transparent 1px);
        background-size: 44px 44px;
        mask-image: radial-gradient(ellipse at center, black 10%, transparent 70%);
      "
    />
    <div class="pointer-events-none absolute top-1/3 left-1/2 size-[480px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-accent/10 blur-3xl" />

    <div class="relative w-full max-w-sm">
      <div class="mb-6 flex items-center gap-3">
        <span class="grid size-10 place-items-center rounded-lg bg-accent/15 text-accent ring-1 ring-accent/30">
          <AppIcon name="shield" :size="20" :stroke="2" />
        </span>
        <div>
          <h1 class="text-base font-semibold">Admin Console</h1>
          <p class="text-xs text-muted">Undangan Digital · akses super admin</p>
        </div>
      </div>

      <form class="card space-y-4 p-5 shadow-2xl shadow-black/40" novalidate @submit.prevent="submit">
        <div v-if="notice" class="flex items-start gap-2 rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-xs text-danger" role="alert">
          <AppIcon name="alert" :size="14" class="mt-px" />
          <span>{{ notice }}</span>
        </div>

        <div>
          <label for="email" class="label">Email</label>
          <input id="email" v-model="email" type="email" class="input" autocomplete="username" placeholder="admin@domain.com" autofocus required />
        </div>
        <div>
          <label for="password" class="label">Password</label>
          <div class="relative">
            <input
              id="password"
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              class="input pr-10"
              autocomplete="current-password"
              required
            />
            <button
              type="button"
              class="absolute top-1/2 right-1 grid size-7 -translate-y-1/2 place-items-center rounded text-muted hover:text-ink"
              :aria-label="showPassword ? 'Sembunyikan password' : 'Tampilkan password'"
              @click="showPassword = !showPassword"
            >
              <AppIcon name="eye" :size="15" />
            </button>
          </div>
        </div>
        <button type="submit" class="btn btn-primary h-9 w-full" :disabled="loading">
          <UiSpinner v-if="loading" :size="14" />
          {{ loading ? 'Memproses…' : 'Masuk' }}
        </button>
      </form>
      <p class="mt-4 text-center text-[11px] text-muted">Aktivitas admin tercatat di audit log.</p>
    </div>
  </div>
</template>
