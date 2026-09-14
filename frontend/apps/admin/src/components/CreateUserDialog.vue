<script setup lang="ts">
import { api, formatRupiah, type Plan, type User } from '@undangan/shared'
import { computed, reactive, ref, watch } from 'vue'
import { toast } from '../lib/toast'
import { copyText, errMsg, fieldErrors, generatePassword, itemsOf } from '../lib/util'
import AppIcon from './AppIcon.vue'
import UiDialog from './UiDialog.vue'
import UiSpinner from './UiSpinner.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; created: [user: User] }>()

const form = reactive({ name: '', email: '', phone: '', password: '', plan_id: '' })
const errors = ref<Record<string, string>>({})
const formError = ref('')
const saving = ref(false)
const showPassword = ref(true)
const plans = ref<Plan[]>([])
const plansLoading = ref(false)

const activePlans = computed(() => plans.value.filter((p) => p.is_active))

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    Object.assign(form, { name: '', email: '', phone: '', password: generatePassword(), plan_id: '' })
    errors.value = {}
    formError.value = ''
    if (!plans.value.length) {
      plansLoading.value = true
      try {
        plans.value = itemsOf(await api.get<{ items: Plan[] }>('/admin/plans'))
      } catch {
        /* opsional */
      } finally {
        plansLoading.value = false
      }
    }
  },
)

function validate() {
  const e: Record<string, string> = {}
  if (!form.name.trim()) e.name = 'Nama wajib diisi.'
  if (!/^\S+@\S+\.\S+$/.test(form.email.trim())) e.email = 'Format email tidak valid.'
  if (form.phone && !/^(\+?62|0)8\d{7,12}$/.test(form.phone.replace(/[\s-]/g, ''))) e.phone = 'Nomor HP tidak valid (contoh 0812…).'
  if (form.password.length < 8) e.password = 'Password minimal 8 karakter.'
  errors.value = e
  return !Object.keys(e).length
}

async function submit() {
  if (!validate()) return
  saving.value = true
  formError.value = ''
  try {
    const res = await api.post<{ user: User }>('/admin/users', {
      name: form.name.trim(),
      email: form.email.trim(),
      phone: form.phone.replace(/[\s-]/g, ''),
      password: form.password,
      ...(form.plan_id ? { plan_id: form.plan_id } : {}),
    })
    await copyText(`Email: ${form.email.trim()}\nPassword: ${form.password}`)
    toast.success(`Customer ${res.user.name} dibuat. Kredensial login disalin ke clipboard.`)
    emit('created', res.user)
  } catch (e) {
    errors.value = fieldErrors(e)
    formError.value = Object.keys(errors.value).length ? '' : errMsg(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UiDialog :open="open" title="Tambah Customer" description="Akun customer baru untuk login di portal customer." :persistent="saving" @close="emit('close')">
    <form id="create-user-form" class="space-y-3.5" novalidate @submit.prevent="submit">
      <p v-if="formError" class="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-xs text-danger" role="alert">{{ formError }}</p>
      <div>
        <label for="cu-name" class="label">Nama lengkap</label>
        <input id="cu-name" v-model="form.name" class="input" :class="errors.name && 'input-error'" autocomplete="off" data-autofocus />
        <p v-if="errors.name" class="field-error">{{ errors.name }}</p>
      </div>
      <div class="grid gap-3.5 sm:grid-cols-2">
        <div>
          <label for="cu-email" class="label">Email</label>
          <input id="cu-email" v-model="form.email" type="email" class="input" :class="errors.email && 'input-error'" autocomplete="off" />
          <p v-if="errors.email" class="field-error">{{ errors.email }}</p>
        </div>
        <div>
          <label for="cu-phone" class="label">No. HP / WhatsApp</label>
          <input id="cu-phone" v-model="form.phone" type="tel" inputmode="tel" class="input" :class="errors.phone && 'input-error'" placeholder="0812…" />
          <p v-if="errors.phone" class="field-error">{{ errors.phone }}</p>
        </div>
      </div>
      <div>
        <label for="cu-password" class="label">Password awal</label>
        <div class="flex gap-2">
          <div class="relative flex-1">
            <input
              id="cu-password"
              v-model="form.password"
              :type="showPassword ? 'text' : 'password'"
              class="input pr-9 font-mono"
              :class="errors.password && 'input-error'"
              autocomplete="new-password"
            />
            <button
              type="button"
              class="absolute top-1/2 right-1 grid size-7 -translate-y-1/2 place-items-center text-muted hover:text-ink"
              :aria-label="showPassword ? 'Sembunyikan password' : 'Tampilkan password'"
              @click="showPassword = !showPassword"
            >
              <AppIcon name="eye" :size="14" />
            </button>
          </div>
          <button type="button" class="btn btn-secondary h-9" title="Buat password acak" @click="form.password = generatePassword()">
            <AppIcon name="refresh" :size="14" /> Acak
          </button>
        </div>
        <p v-if="errors.password" class="field-error">{{ errors.password }}</p>
        <p v-else class="hint">Kredensial akan disalin ke clipboard setelah akun dibuat — kirimkan ke customer.</p>
      </div>
      <div>
        <label for="cu-plan" class="label">Beri langganan (opsional)</label>
        <select id="cu-plan" v-model="form.plan_id" class="input" :disabled="plansLoading">
          <option value="">— Tanpa langganan —</option>
          <option v-for="p in activePlans" :key="p.id" :value="p.id">{{ p.name }} · {{ p.duration_days }} hari · {{ formatRupiah(p.price) }}</option>
        </select>
        <p v-if="errors.plan_id" class="field-error">{{ errors.plan_id }}</p>
        <p v-else class="hint">Langganan langsung aktif tanpa order pembayaran (tercatat di audit log).</p>
      </div>
    </form>
    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="saving" @click="emit('close')">Batal</button>
      <button type="submit" form="create-user-form" class="btn btn-primary" :disabled="saving">
        <UiSpinner v-if="saving" :size="14" /> Buat akun
      </button>
    </template>
  </UiDialog>
</template>
