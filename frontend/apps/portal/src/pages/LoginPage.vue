<script setup lang="ts">
import { ApiError, errorMessage } from '@undangan/shared'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiField from '@undangan/shared/components/ui/UiField.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiInput from '@undangan/shared/components/ui/UiInput.vue'
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthShell from '../components/AuthShell.vue'
import { APP_NAME } from '../lib/brand'
import { login, logout } from '../lib/auth'

const route = useRoute()
const router = useRouter()

const form = reactive({ email: '', password: '' })
const showPw = ref(false)
const loading = ref(false)
const error = ref('')
const fields = ref<Record<string, string>>({})

async function submit() {
  error.value = ''
  fields.value = {}
  if (!form.email || !form.password) {
    error.value = 'Email dan kata sandi wajib diisi.'
    return
  }
  loading.value = true
  try {
    const user = await login(form.email.trim(), form.password)
    if (user.role !== 'customer') {
      await logout()
      error.value = 'Akun ini bukan akun customer. Silakan masuk melalui portal admin.'
      return
    }
    const next = typeof route.query.next === 'string' && route.query.next.startsWith('/') ? route.query.next : '/'
    router.replace(next)
  } catch (e) {
    if (e instanceof ApiError) fields.value = e.fields
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AuthShell title="Selamat datang" :subtitle="`Masuk ke ${APP_NAME} untuk mengelola undangan digital Anda.`">
    <form class="space-y-4" novalidate @submit.prevent="submit">
      <div v-if="error" class="flex items-start gap-2 rounded-xl bg-danger/10 px-3 py-2.5 text-sm text-danger" role="alert">
        <UiIcon name="alert" :size="16" class="mt-0.5" />
        <span>{{ error }}</span>
      </div>
      <UiField label="Email" :error="fields.email">
        <UiInput v-model="form.email" type="email" autocomplete="email" placeholder="nama@email.com" :invalid="!!fields.email" />
      </UiField>
      <UiField label="Kata sandi" :error="fields.password">
        <div class="relative">
          <UiInput
            v-model="form.password"
            :type="showPw ? 'text' : 'password'"
            autocomplete="current-password"
            placeholder="••••••••"
            class="pr-10!"
            :invalid="!!fields.password"
          />
          <button
            type="button"
            class="absolute top-1/2 right-2 -translate-y-1/2 rounded-md p-1.5 text-muted hover:text-ink"
            :aria-label="showPw ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'"
            @click="showPw = !showPw"
          >
            <UiIcon :name="showPw ? 'eye-off' : 'eye'" :size="16" />
          </button>
        </div>
      </UiField>
      <UiButton type="submit" block :loading="loading" class="h-11!">Masuk</UiButton>
    </form>
    <template #footer>
      Belum punya akun?
      <RouterLink :to="{ path: '/daftar', query: route.query }" class="font-semibold text-accent hover:underline">Daftar sekarang</RouterLink>
    </template>
  </AuthShell>
</template>
