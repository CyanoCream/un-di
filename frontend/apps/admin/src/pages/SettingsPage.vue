<script setup lang="ts">
import { api, formatRupiah, type PaymentSettings } from '@undangan/shared'
import ImageUpload from '@undangan/shared/components/ImageUpload.vue'
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import PageHeader from '../components/PageHeader.vue'
import UiSpinner from '../components/UiSpinner.vue'
import UiState from '../components/UiState.vue'
import { confirmDialog } from '../lib/confirm'
import { toast } from '../lib/toast'
import { errMsg, fieldErrors } from '../lib/util'

const form = reactive<PaymentSettings>({ qris_image: '', merchant_name: '', instructions: '', admin_whatsapp: '' })
const saved = ref('')
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const errors = ref<Record<string, string>>({})

const snapshot = () => JSON.stringify(form)
const dirty = computed(() => !loading.value && !error.value && snapshot() !== saved.value)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const s = await api.get<PaymentSettings>('/admin/settings/payment')
    Object.assign(form, { qris_image: s.qris_image ?? '', merchant_name: s.merchant_name ?? '', instructions: s.instructions ?? '', admin_whatsapp: s.admin_whatsapp ?? '' })
    saved.value = snapshot()
  } catch (e) {
    error.value = errMsg(e)
  } finally {
    loading.value = false
  }
}
onMounted(load)

function normalizeWa(v: string) {
  const d = v.replace(/\D/g, '')
  if (d.startsWith('0')) return '62' + d.slice(1)
  if (d.startsWith('8')) return '62' + d
  return d
}

function validate() {
  const e: Record<string, string> = {}
  form.admin_whatsapp = normalizeWa(form.admin_whatsapp)
  if (!form.qris_image) e.qris_image = 'Unggah gambar QRIS.'
  if (!form.merchant_name.trim()) e.merchant_name = 'Nama merchant wajib diisi.'
  if (!/^628\d{7,12}$/.test(form.admin_whatsapp)) e.admin_whatsapp = 'Format 628xxxxxxxxx (10–15 digit).'
  errors.value = e
  return !Object.keys(e).length
}

async function save() {
  if (!validate()) {
    toast.error('Periksa kembali isian yang ditandai.')
    return
  }
  saving.value = true
  try {
    const s = await api.put<PaymentSettings>('/admin/settings/payment', {
      qris_image: form.qris_image,
      merchant_name: form.merchant_name.trim(),
      instructions: form.instructions.trim(),
      admin_whatsapp: form.admin_whatsapp,
    })
    if (s) Object.assign(form, s)
    saved.value = snapshot()
    toast.success('Pengaturan pembayaran disimpan.')
  } catch (e) {
    errors.value = fieldErrors(e)
    toast.error(errMsg(e))
  } finally {
    saving.value = false
  }
}

function revert() {
  if (saved.value) Object.assign(form, JSON.parse(saved.value))
  errors.value = {}
}

onBeforeRouteLeave(async () => {
  if (!dirty.value) return true
  return confirmDialog({ title: 'Buang perubahan?', message: 'Perubahan pengaturan pembayaran belum disimpan.', confirmText: 'Buang', tone: 'danger' })
})
function beforeUnload(e: BeforeUnloadEvent) {
  if (dirty.value) e.preventDefault()
}
onMounted(() => window.addEventListener('beforeunload', beforeUnload))
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))

// ---- Pratinjau checkout ----
const SAMPLE = { plan: 'Premium', price: 149_000, code: 237 }
const deadline = computed(() =>
  new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(Date.now() + 86_400_000)),
)
const waPreview = computed(() => (form.admin_whatsapp ? `wa.me/${normalizeWa(form.admin_whatsapp)}` : 'nomor WA belum diisi'))
</script>

<template>
  <div>
    <PageHeader title="Pengaturan Pembayaran" subtitle="QRIS statis & kontak admin yang tampil di halaman checkout customer.">
      <template #actions>
        <span v-if="dirty" class="inline-flex items-center gap-1.5 text-xs text-warning"><span class="size-1.5 rounded-full bg-warning" /> Belum disimpan</span>
        <button v-if="dirty" type="button" class="btn btn-ghost" :disabled="saving" @click="revert">Batalkan</button>
        <button type="button" class="btn btn-primary" :disabled="saving || !dirty" @click="save">
          <UiSpinner v-if="saving" :size="14" /> Simpan
        </button>
      </template>
    </PageHeader>

    <UiState :loading="loading" :error="error" :empty="loading" @retry="load">
      <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_380px]">
        <form class="card space-y-5 p-5" novalidate @submit.prevent="save">
          <div>
            <span class="label">Gambar QRIS</span>
            <ImageUpload v-model="form.qris_image" aspect="3 / 4" :max-size-mb="5" />
            <p v-if="errors.qris_image" class="field-error">{{ errors.qris_image }}</p>
            <p v-else class="hint">Gunakan QRIS statis resmi dari bank/penyedia. Pastikan kode QR terbaca jelas.</p>
          </div>

          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label for="s-merchant" class="label">Nama merchant</label>
              <input id="s-merchant" v-model="form.merchant_name" class="input" :class="errors.merchant_name && 'input-error'" placeholder="UNDANGAN DIGITAL" />
              <p v-if="errors.merchant_name" class="field-error">{{ errors.merchant_name }}</p>
              <p v-else class="hint">Samakan dengan nama yang muncul saat QR dipindai.</p>
            </div>
            <div>
              <label for="s-wa" class="label">WhatsApp admin</label>
              <input
                id="s-wa"
                v-model="form.admin_whatsapp"
                type="tel"
                inputmode="numeric"
                class="input num"
                :class="errors.admin_whatsapp && 'input-error'"
                placeholder="6281234567890"
                @blur="form.admin_whatsapp = normalizeWa(form.admin_whatsapp)"
              />
              <p v-if="errors.admin_whatsapp" class="field-error">{{ errors.admin_whatsapp }}</p>
              <p v-else class="hint">Format 628… — dipakai tombol “Konfirmasi via WhatsApp”.</p>
            </div>
          </div>

          <div>
            <label for="s-instr" class="label">Instruksi pembayaran</label>
            <textarea
              id="s-instr"
              v-model="form.instructions"
              rows="6"
              maxlength="2000"
              class="input"
              placeholder="1. Buka aplikasi m-banking / e-wallet&#10;2. Pindai kode QRIS di atas&#10;3. Masukkan nominal PERSIS sesuai tagihan&#10;4. Unggah bukti pembayaran"
            />
            <p class="hint num">{{ form.instructions.length }}/2000 · baris baru dipertahankan</p>
          </div>
        </form>

        <!-- Pratinjau -->
        <aside aria-label="Pratinjau checkout customer">
          <div class="sticky top-20">
            <p class="mb-2 flex items-center gap-1.5 text-xs text-muted"><AppIcon name="eye" :size="13" /> Pratinjau checkout customer (contoh)</p>
            <div class="mx-auto w-full max-w-[340px] rounded-[2rem] border border-line bg-surface-2 p-2.5 shadow-2xl shadow-black/50">
              <div class="max-h-[640px] overflow-y-auto rounded-[1.5rem] bg-[#FBF6F0] px-4 py-5 text-[#3B2F2F]" style="font-family: Georgia, 'Times New Roman', serif">
                <p class="text-center text-[11px] tracking-[0.2em] text-[#B0707A] uppercase" style="font-family: var(--font-sans)">Pembayaran</p>
                <h3 class="mt-1 text-center text-lg">Paket {{ SAMPLE.plan }}</h3>

                <div class="mt-4 rounded-2xl bg-white p-3 shadow-sm ring-1 ring-[#EADDD3]">
                  <div class="mx-auto flex aspect-[3/4] w-full max-w-[220px] items-center justify-center overflow-hidden rounded-lg bg-[#F4ECE4]">
                    <img v-if="form.qris_image" :src="form.qris_image" alt="QRIS" class="h-full w-full object-contain" />
                    <span v-else class="px-4 text-center text-xs text-[#9A8A80]" style="font-family: var(--font-sans)">Gambar QRIS belum diunggah</span>
                  </div>
                  <p class="mt-2 text-center text-sm font-semibold" style="font-family: var(--font-sans)">{{ form.merchant_name || 'Nama merchant' }}</p>
                </div>

                <div class="mt-4 text-center" style="font-family: var(--font-sans)">
                  <p class="text-[11px] text-[#8C7B72]">Transfer tepat sebesar</p>
                  <p class="num mt-0.5 text-2xl font-semibold text-[#3B2F2F]">
                    {{ formatRupiah(SAMPLE.price + SAMPLE.code).slice(0, -3) }}<span class="text-[#B0707A]">{{ formatRupiah(SAMPLE.price + SAMPLE.code).slice(-3) }}</span>
                  </p>
                  <span class="mt-1.5 inline-flex items-center gap-1 rounded-full bg-[#F3E3E1] px-3 py-1 text-[11px] text-[#9C5B65]"><AppIcon name="copy" :size="11" /> Salin nominal</span>
                  <p class="mt-2 text-[11px] text-[#8C7B72]">3 digit terakhir adalah kode unik. Bayar sebelum <b>{{ deadline }}</b>.</p>
                </div>

                <div v-if="form.instructions.trim()" class="mt-4 rounded-xl bg-white/70 p-3 text-xs leading-relaxed whitespace-pre-line ring-1 ring-[#EADDD3]" style="font-family: var(--font-sans)">
                  {{ form.instructions }}
                </div>

                <div class="mt-4 space-y-2" style="font-family: var(--font-sans)">
                  <div class="flex h-10 items-center justify-center gap-1.5 rounded-full bg-[#B0707A] text-sm font-medium text-white"><AppIcon name="upload" :size="14" /> Unggah bukti bayar</div>
                  <div class="flex h-10 items-center justify-center gap-1.5 rounded-full border border-[#B0707A] text-sm font-medium text-[#9C5B65]">
                    <AppIcon name="message" :size="14" /> Konfirmasi via WhatsApp
                  </div>
                  <p class="text-center text-[10px] text-[#9A8A80]">{{ waPreview }}</p>
                </div>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </UiState>
  </div>
</template>
