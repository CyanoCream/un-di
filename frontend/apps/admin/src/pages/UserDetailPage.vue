<script setup lang="ts">
import { api, formatDate, formatRupiah, type InvitationSummary, type Plan, type Subscription, type User } from '@undangan/shared'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import CopyButton from '../components/CopyButton.vue'
import CreateInvitationDialog from '../components/CreateInvitationDialog.vue'
import UiBadge from '../components/UiBadge.vue'
import UiDialog from '../components/UiDialog.vue'
import UiSpinner from '../components/UiSpinner.vue'
import UiState from '../components/UiState.vue'
import UiToggle from '../components/UiToggle.vue'
import { confirmDialog } from '../lib/confirm'
import { INVITATION_STATUS, ROLE_LABEL, SUBSCRIPTION_STATUS } from '../lib/labels'
import { toast } from '../lib/toast'
import { daysUntil, errMsg, fieldErrors, formatNumber, initials, itemsOf } from '../lib/util'

const route = useRoute()
const id = computed(() => String(route.params.id))

const user = ref<User | null>(null)
const subscription = ref<Subscription | null>(null)
const invitations = ref<InvitationSummary[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<{ user: User; subscription: Subscription | null; invitations: InvitationSummary[] }>(`/admin/users/${id.value}`)
    user.value = res.user
    subscription.value = res.subscription
    invitations.value = res.invitations ?? []
    profile.name = res.user.name
    profile.phone = res.user.phone ?? ''
  } catch (e) {
    error.value = errMsg(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ---- Profil ----
const profile = reactive({ name: '', phone: '' })
const profileErrors = ref<Record<string, string>>({})
const savingProfile = ref(false)
const profileDirty = computed(() => !!user.value && (profile.name !== user.value.name || profile.phone !== (user.value.phone ?? '')))

async function patchUser(body: { name: string; phone: string; is_suspended: boolean }) {
  const res = await api.patch<{ user: User }>(`/admin/users/${id.value}`, body)
  user.value = res.user
  return res.user
}

async function saveProfile() {
  if (!user.value) return
  profileErrors.value = {}
  if (!profile.name.trim()) {
    profileErrors.value.name = 'Nama wajib diisi.'
    return
  }
  savingProfile.value = true
  try {
    const u = await patchUser({ name: profile.name.trim(), phone: profile.phone.trim(), is_suspended: user.value.is_suspended })
    profile.name = u.name
    profile.phone = u.phone ?? ''
    toast.success('Profil disimpan.')
  } catch (e) {
    profileErrors.value = fieldErrors(e)
    if (!Object.keys(profileErrors.value).length) toast.error(errMsg(e))
  } finally {
    savingProfile.value = false
  }
}

// ---- Suspend ----
const suspending = ref(false)
async function toggleSuspend(next: boolean) {
  const u = user.value
  if (!u) return
  const ok = await confirmDialog(
    next
      ? {
          title: `Tangguhkan ${u.name}?`,
          message: 'Customer tidak bisa login ke portal sampai penangguhan dicabut.',
          details: ['Sesi login yang aktif akan diakhiri.', 'Undangan yang sudah terbit tetap dapat diakses tamu kecuali ditangguhkan terpisah.', 'Tindakan tercatat di audit log.'],
          confirmText: 'Tangguhkan',
          tone: 'danger',
        }
      : { title: `Aktifkan kembali ${u.name}?`, message: 'Customer dapat login kembali ke portal.', confirmText: 'Aktifkan', tone: 'success' },
  )
  if (!ok) return
  suspending.value = true
  try {
    await patchUser({ name: u.name, phone: u.phone ?? '', is_suspended: next })
    toast.success(next ? 'Akun ditangguhkan.' : 'Akun diaktifkan kembali.')
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    suspending.value = false
  }
}

// ---- Reset password ----
const resetting = ref(false)
const tempPassword = ref<string | null>(null)
async function resetPassword() {
  const u = user.value
  if (!u) return
  const ok = await confirmDialog({
    title: 'Reset password?',
    message: `Password ${u.name} akan diganti dengan password sementara.`,
    details: ['Password lama langsung tidak berlaku.', 'Password sementara hanya ditampilkan sekali — kirimkan ke customer.'],
    confirmText: 'Reset password',
    tone: 'danger',
  })
  if (!ok) return
  resetting.value = true
  try {
    const res = await api.post<{ password: string }>(`/admin/users/${id.value}/reset-password`)
    tempPassword.value = res.password
  } catch (e) {
    toast.error(errMsg(e))
  } finally {
    resetting.value = false
  }
}

// ---- Langganan ----
const subOpen = ref(false)
const plans = ref<Plan[]>([])
const subForm = reactive({ plan_id: '', custom: false, days: 30 })
const subError = ref('')
const grantingSub = ref(false)
const activePlans = computed(() => plans.value.filter((p) => p.is_active))
const selectedPlan = computed(() => plans.value.find((p) => p.id === subForm.plan_id) ?? null)
const subIsRunning = computed(() => !!subscription.value && (subscription.value.status === 'active' || subscription.value.status === 'grace'))
const grantDays = computed(() => (subForm.custom ? subForm.days : (selectedPlan.value?.duration_days ?? 0)))
const projectedEnd = computed(() => {
  if (!grantDays.value) return null
  const base = subIsRunning.value ? Math.max(Date.now(), new Date(subscription.value!.ends_at).getTime()) : Date.now()
  return new Date(base + grantDays.value * 86_400_000).toISOString()
})

async function openSubDialog() {
  subError.value = ''
  subForm.custom = false
  subOpen.value = true
  if (!plans.value.length) {
    try {
      plans.value = itemsOf(await api.get<{ items: Plan[] }>('/admin/plans'))
    } catch (e) {
      subError.value = errMsg(e)
    }
  }
  subForm.plan_id = subscription.value?.plan_id && activePlans.value.some((p) => p.id === subscription.value?.plan_id) ? subscription.value.plan_id : (activePlans.value[0]?.id ?? '')
  subForm.days = selectedPlan.value?.duration_days ?? 30
}

async function grantSubscription() {
  subError.value = ''
  if (!subForm.plan_id) {
    subError.value = 'Pilih paket.'
    return
  }
  if (subForm.custom && (!Number.isInteger(subForm.days) || subForm.days < 1 || subForm.days > 3650)) {
    subError.value = 'Jumlah hari harus 1–3650.'
    return
  }
  grantingSub.value = true
  try {
    const sub = await api.post<Subscription>(`/admin/users/${id.value}/subscriptions`, {
      plan_id: subForm.plan_id,
      ...(subForm.custom ? { days: subForm.days } : {}),
    })
    subscription.value = sub
    subOpen.value = false
    toast.success(`Langganan ${sub.plan_name} aktif s.d. ${formatDate(sub.ends_at)}.`)
  } catch (e) {
    subError.value = errMsg(e)
  } finally {
    grantingSub.value = false
  }
}

const subDays = computed(() => {
  const s = subscription.value
  if (!s) return null
  return s.status === 'grace' ? daysUntil(s.grace_ends_at) : daysUntil(s.ends_at)
})

// ---- Undangan ----
const createInvOpen = ref(false)
</script>

<template>
  <div>
    <RouterLink to="/users" class="mb-3 inline-flex items-center gap-1 text-xs text-muted hover:text-ink">
      <AppIcon name="chevron-left" :size="14" /> Pengguna
    </RouterLink>

    <UiState :loading="loading" :error="error" :empty="!user" empty-title="Pengguna tidak ditemukan" @retry="load">
      <div v-if="user">
        <!-- Header -->
        <div class="mb-5 flex flex-wrap items-center gap-4">
          <span class="grid size-12 place-items-center rounded-full bg-accent/15 text-base font-semibold text-accent ring-1 ring-accent/30">{{ initials(user.name) }}</span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h1 class="truncate text-lg font-semibold tracking-tight">{{ user.name }}</h1>
              <UiBadge :tone="user.role === 'super_admin' ? 'accent' : 'neutral'" :dot="false">{{ ROLE_LABEL[user.role] }}</UiBadge>
              <UiBadge v-if="user.is_suspended" tone="danger">Ditangguhkan</UiBadge>
            </div>
            <p class="mt-0.5 flex flex-wrap items-center gap-x-3 text-xs text-muted">
              <span class="inline-flex items-center gap-1">{{ user.email }} <CopyButton :text="user.email" /></span>
              <span>Terdaftar {{ formatDate(user.created_at) }}</span>
            </p>
          </div>
        </div>

        <div class="grid gap-5 lg:grid-cols-3">
          <!-- Kolom kiri -->
          <div class="space-y-5 lg:col-span-1">
            <section class="card" aria-labelledby="profile-title">
              <div class="card-header"><h2 id="profile-title" class="card-title">Profil</h2></div>
              <form class="space-y-3 p-4" @submit.prevent="saveProfile">
                <div>
                  <label for="p-name" class="label">Nama</label>
                  <input id="p-name" v-model="profile.name" class="input" :class="profileErrors.name && 'input-error'" />
                  <p v-if="profileErrors.name" class="field-error">{{ profileErrors.name }}</p>
                </div>
                <div>
                  <label for="p-email" class="label">Email</label>
                  <input id="p-email" :value="user.email" class="input" disabled />
                </div>
                <div>
                  <label for="p-phone" class="label">No. HP / WhatsApp</label>
                  <input id="p-phone" v-model="profile.phone" type="tel" class="input" :class="profileErrors.phone && 'input-error'" />
                  <p v-if="profileErrors.phone" class="field-error">{{ profileErrors.phone }}</p>
                </div>
                <div class="flex justify-end">
                  <button type="submit" class="btn btn-primary" :disabled="!profileDirty || savingProfile">
                    <UiSpinner v-if="savingProfile" :size="14" /> Simpan
                  </button>
                </div>
              </form>
            </section>

            <section class="card" aria-labelledby="security-title">
              <div class="card-header"><h2 id="security-title" class="card-title">Akses akun</h2></div>
              <div class="divide-y divide-line/60">
                <div class="flex items-center justify-between gap-3 px-4 py-3">
                  <div>
                    <p class="text-[13px] font-medium">Tangguhkan akun</p>
                    <p class="text-xs text-muted">Blokir login customer.</p>
                  </div>
                  <UiToggle
                    :model-value="user.is_suspended"
                    tone="danger"
                    label="Tangguhkan akun"
                    :disabled="suspending || user.role === 'super_admin'"
                    @update:model-value="toggleSuspend"
                  />
                </div>
                <div class="flex items-center justify-between gap-3 px-4 py-3">
                  <div>
                    <p class="text-[13px] font-medium">Reset password</p>
                    <p class="text-xs text-muted">Buat password sementara.</p>
                  </div>
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="resetting" @click="resetPassword">
                    <UiSpinner v-if="resetting" :size="12" /><AppIcon v-else name="key" :size="13" /> Reset
                  </button>
                </div>
              </div>
            </section>
          </div>

          <!-- Kolom kanan -->
          <div class="space-y-5 lg:col-span-2">
            <section class="card" aria-labelledby="sub-title">
              <div class="card-header">
                <h2 id="sub-title" class="card-title">Langganan</h2>
                <button type="button" class="btn btn-secondary btn-sm" @click="openSubDialog">
                  <AppIcon name="plus" :size="13" /> {{ subIsRunning ? 'Perpanjang' : 'Beri' }} Langganan
                </button>
              </div>
              <div v-if="!subscription" class="flex items-center gap-3 px-4 py-5 text-sm text-muted">
                <AppIcon name="card" :size="18" /> Customer ini belum memiliki langganan.
              </div>
              <div v-else class="grid gap-4 p-4 sm:grid-cols-4">
                <div class="sm:col-span-1">
                  <p class="text-xs text-muted">Paket</p>
                  <p class="mt-0.5 font-semibold">{{ subscription.plan_name }}</p>
                  <UiBadge class="mt-1.5" :tone="SUBSCRIPTION_STATUS[subscription.status]?.tone">{{ SUBSCRIPTION_STATUS[subscription.status]?.label }}</UiBadge>
                </div>
                <div>
                  <p class="text-xs text-muted">Periode</p>
                  <p class="num mt-0.5 text-[13px]">{{ formatDate(subscription.starts_at) }}</p>
                  <p class="num text-[13px]">s.d. {{ formatDate(subscription.ends_at) }}</p>
                </div>
                <div>
                  <p class="text-xs text-muted">{{ subscription.status === 'grace' ? 'Sisa masa tenggang' : 'Sisa hari' }}</p>
                  <p
                    class="num mt-0.5 text-xl font-semibold"
                    :class="subDays === null || subDays < 0 ? 'text-muted' : subDays <= 7 ? 'text-warning' : 'text-ink'"
                  >
                    {{ subDays === null ? '-' : Math.max(0, subDays) }}
                  </p>
                  <p class="num text-xs text-muted">Tenggang s.d. {{ formatDate(subscription.grace_ends_at) }}</p>
                </div>
                <div>
                  <p class="text-xs text-muted">Kuota</p>
                  <p class="num mt-0.5 text-[13px]">{{ formatNumber(subscription.max_invitations) }} undangan</p>
                  <p class="num text-[13px]">{{ formatNumber(subscription.max_guests) }} tamu / undangan</p>
                </div>
              </div>
            </section>

            <section class="card" aria-labelledby="inv-title">
              <div class="card-header">
                <h2 id="inv-title" class="card-title">Undangan <span class="num ml-1 text-muted">{{ invitations.length }}</span></h2>
                <button type="button" class="btn btn-secondary btn-sm" :disabled="user.role !== 'customer'" @click="createInvOpen = true">
                  <AppIcon name="plus" :size="13" /> Buat Undangan untuk user ini
                </button>
              </div>
              <UiState :empty="!invitations.length" empty-title="Belum ada undangan" empty-icon="mail" compact>
                <div class="table-wrap">
                  <table class="data-table">
                    <thead>
                      <tr>
                        <th>Judul</th>
                        <th>Status</th>
                        <th>Tema</th>
                        <th>Subdomain</th>
                        <th>Acara</th>
                        <th class="text-right">Tamu</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="inv in invitations" :key="inv.id" class="row-link" @click="$router.push(`/invitations/${inv.id}`)">
                        <td class="max-w-56 truncate font-medium">{{ inv.title }}</td>
                        <td><UiBadge :tone="INVITATION_STATUS[inv.status]?.tone">{{ INVITATION_STATUS[inv.status]?.label ?? inv.status }}</UiBadge></td>
                        <td class="text-muted">{{ inv.theme ?? '-' }}</td>
                        <td>
                          <a v-if="inv.url" :href="inv.url" target="_blank" rel="noopener" class="font-mono text-xs text-accent hover:underline" @click.stop>{{ inv.subdomain }}</a>
                          <span v-else class="text-muted">-</span>
                        </td>
                        <td class="num text-muted">{{ formatDate(inv.event_date) }}</td>
                        <td class="num text-right">{{ formatNumber(inv.guest_count) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </UiState>
            </section>
          </div>
        </div>
      </div>
    </UiState>

    <!-- Password sementara -->
    <UiDialog :open="!!tempPassword" title="Password sementara" size="sm" @close="tempPassword = null">
      <div class="space-y-3">
        <div class="flex items-start gap-2 rounded-md border border-warning/30 bg-warning/10 px-3 py-2 text-xs text-warning">
          <AppIcon name="alert" :size="14" class="mt-px" /> Password hanya ditampilkan sekali. Salin dan kirimkan ke customer sekarang.
        </div>
        <div class="flex items-center gap-2 rounded-md border border-line bg-canvas px-3 py-2.5">
          <code class="flex-1 font-mono text-base tracking-wider break-all select-all">{{ tempPassword }}</code>
          <CopyButton :text="tempPassword ?? ''" label="Salin" />
        </div>
      </div>
      <template #footer>
        <button type="button" class="btn btn-primary" @click="tempPassword = null">Selesai</button>
      </template>
    </UiDialog>

    <!-- Beri / perpanjang langganan -->
    <UiDialog
      :open="subOpen"
      :title="subIsRunning ? 'Perpanjang Langganan' : 'Beri Langganan'"
      :description="user ? `${user.name} · tanpa order pembayaran` : undefined"
      :persistent="grantingSub"
      @close="subOpen = false"
    >
      <form id="grant-sub-form" class="space-y-4" @submit.prevent="grantSubscription">
        <p v-if="subError" class="rounded-md border border-danger/30 bg-danger/10 px-3 py-2 text-xs text-danger" role="alert">{{ subError }}</p>
        <div>
          <label for="gs-plan" class="label">Paket</label>
          <select id="gs-plan" v-model="subForm.plan_id" class="input">
            <option v-for="p in activePlans" :key="p.id" :value="p.id">{{ p.name }} · {{ p.duration_days }} hari · {{ formatRupiah(p.price) }}</option>
          </select>
          <p v-if="selectedPlan" class="hint">Kuota {{ selectedPlan.max_invitations }} undangan, {{ formatNumber(selectedPlan.max_guests) }} tamu, tenggang {{ selectedPlan.grace_days }} hari.</p>
        </div>
        <div class="rounded-md border border-line p-3">
          <label class="flex cursor-pointer items-center gap-2 text-[13px]">
            <input v-model="subForm.custom" type="checkbox" class="accent-[var(--color-accent)]" />
            Gunakan jumlah hari khusus
          </label>
          <div v-if="subForm.custom" class="mt-2.5 flex items-center gap-2">
            <input v-model.number="subForm.days" type="number" min="1" max="3650" class="input num w-28" aria-label="Jumlah hari" />
            <span class="text-xs text-muted">hari</span>
          </div>
        </div>
        <div class="rounded-md bg-canvas/60 px-3 py-2.5 text-xs text-muted">
          <template v-if="subIsRunning && subscription">
            Langganan berjalan (berakhir {{ formatDate(subscription.ends_at) }}) diperpanjang
            <b class="num text-ink">{{ grantDays }} hari</b>
          </template>
          <template v-else>
            Langganan baru aktif mulai sekarang selama <b class="num text-ink">{{ grantDays }} hari</b>
          </template>
          <span v-if="projectedEnd"> → berakhir ±<b class="num text-ink">{{ formatDate(projectedEnd) }}</b>.</span>
        </div>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="grantingSub" @click="subOpen = false">Batal</button>
        <button type="submit" form="grant-sub-form" class="btn btn-primary" :disabled="grantingSub || !subForm.plan_id">
          <UiSpinner v-if="grantingSub" :size="14" /> {{ subIsRunning ? 'Perpanjang' : 'Aktifkan' }}
        </button>
      </template>
    </UiDialog>

    <CreateInvitationDialog :open="createInvOpen" :preset-user="user" @close="createInvOpen = false" />
  </div>
</template>
