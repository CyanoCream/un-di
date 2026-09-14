<script setup lang="ts">
import { ApiError, errorMessage, toast } from '@undangan/shared'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiField from '@undangan/shared/components/ui/UiField.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import UiInput from '@undangan/shared/components/ui/UiInput.vue'
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthShell from '../components/AuthShell.vue'
import { APP_NAME } from '../lib/brand'
import { firstName, register } from '../lib/auth'
import { getPendingTheme, prettySlug, savePendingTheme } from '../lib/pendingTheme'

const route = useRoute()
const router = useRouter()

// ?tema=<slug> dari landing page: disimpan (juga oleh router) agar disorot saat memilih tema nanti.
savePendingTheme(route.query.tema)
const pendingTheme = computed(() => (typeof route.query.tema === 'string' ? getPendingTheme() : null))

const form = reactive({ name: '', email: '', phone: '', password: '', password2: '' })
const showPw = ref(false)
const loading = ref(false)
const error = ref('')
const fields = ref<Record<string, string>>({})

function validate() {
  const f: Record<string, string> = {}
  if (!form.name.trim()) f.name = 'Nama wajib diisi'
  if (!/^\S+@\S+\.\S+$/.test(form.email.trim())) f.email = 'Email tidak valid'
  if (form.phone && !/^(\+?62|0)8\d{7,12}$/.test(form.phone.replace(/[\s-]/g, ''))) f.phone = 'Nomor HP tidak valid, mis. 081234567890'
  if (form.password.length < 8) f.password = 'Minimal 8 karakter'
  if (form.password2 !== form.password) f.password2 = 'Konfirmasi kata sandi tidak sama'
  fields.value = f
  return Object.keys(f).length === 0
}

async function submit() {
  error.value = ''
  if (!validate()) return
  loading.value = true
  try {
    const user = await register({
      name: form.name.trim(),
      email: form.email.trim(),
      phone: form.phone.replace(/[\s-]/g, ''),
      password: form.password,
    })
    toast.success(`Selamat datang, ${firstName(user)}! Akun Anda berhasil dibuat.`)
    const next = typeof route.query.next === 'string' && route.query.next.startsWith('/') ? route.query.next : '/paket'
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
  <AuthShell title="Buat akun" :subtitle="`Mulai buat undangan digital yang elegan bersama ${APP_NAME} dalam hitungan menit.`">
    <form class="space-y-4" novalidate @submit.prevent="submit">
      <div v-if="pendingTheme" class="flex items-start gap-2 rounded-xl border border-accent/25 bg-accent/8 px-3 py-2.5 text-sm text-ink">
        <UiIcon name="palette" :size="16" class="mt-0.5 shrink-0 text-accent" />
        <span>Tema <strong>{{ prettySlug(pendingTheme) }}</strong> akan kami tandai saat Anda memilih tema undangan nanti.</span>
      </div>
      <div v-if="error" class="flex items-start gap-2 rounded-xl bg-danger/10 px-3 py-2.5 text-sm text-danger" role="alert">
        <UiIcon name="alert" :size="16" class="mt-0.5" />
        <span>{{ error }}</span>
      </div>
      <UiField label="Nama lengkap" :error="fields.name">
        <UiInput v-model="form.name" autocomplete="name" placeholder="Nama Anda" :invalid="!!fields.name" />
      </UiField>
      <UiField label="Email" :error="fields.email">
        <UiInput v-model="form.email" type="email" autocomplete="email" placeholder="nama@email.com" :invalid="!!fields.email" />
      </UiField>
      <UiField label="No. WhatsApp" :error="fields.phone" hint="Untuk konfirmasi pembayaran">
        <UiInput v-model="form.phone" type="tel" inputmode="tel" autocomplete="tel" placeholder="081234567890" :invalid="!!fields.phone" />
      </UiField>
      <div class="grid gap-4 sm:grid-cols-2">
        <UiField label="Kata sandi" :error="fields.password">
          <UiInput v-model="form.password" :type="showPw ? 'text' : 'password'" autocomplete="new-password" :invalid="!!fields.password" />
        </UiField>
        <UiField label="Ulangi kata sandi" :error="fields.password2">
          <UiInput v-model="form.password2" :type="showPw ? 'text' : 'password'" autocomplete="new-password" :invalid="!!fields.password2" />
        </UiField>
      </div>
      <label class="flex items-center gap-2 text-sm text-muted">
        <input v-model="showPw" type="checkbox" class="size-4 accent-accent" /> Tampilkan kata sandi
      </label>
      <UiButton type="submit" block :loading="loading" class="h-11!">Daftar</UiButton>
    </form>
    <template #footer>
      Sudah punya akun?
      <RouterLink :to="{ path: '/masuk', query: route.query }" class="font-semibold text-accent hover:underline">Masuk</RouterLink>
    </template>
  </AuthShell>
</template>
