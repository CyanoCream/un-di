<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ApiError, api } from '../api'
import { toast } from '../composables/useToast'
import type { Invitation } from '../types'
import { errorMessage, sanitizeSubdomain, validateSubdomain } from '../utils/misc'
import UiButton from './ui/UiButton.vue'
import UiIcon from './ui/UiIcon.vue'
import UiSpinner from './ui/UiSpinner.vue'

const props = withDefaults(
  defineProps<{
    invitation: Invitation
    editable: boolean
    /** mis. "undangan.id" — ditampilkan sebagai akhiran. */
    baseDomainHint?: string
    /** Pesan saat tidak bisa diubah. */
    lockedHint?: string
  }>(),
  { baseDomainHint: undefined, lockedHint: 'Subdomain tidak dapat diubah saat undangan sudah dipublikasikan.' },
)

const emit = defineEmits<{ saved: [inv: Invitation] }>()

const value = ref(props.invitation.subdomain ?? '')
type CheckState = 'idle' | 'invalid' | 'checking' | 'available' | 'taken' | 'same' | 'error'
const state = ref<CheckState>('idle')
const reason = ref('')
const saving = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined
let reqSeq = 0

watch(
  () => props.invitation.subdomain,
  (s) => {
    value.value = s ?? ''
    state.value = s ? 'same' : 'idle'
  },
)

const suffix = computed(() => {
  if (props.baseDomainHint) return props.baseDomainHint.replace(/^\./, '')
  const url = props.invitation.url
  if (url && props.invitation.subdomain) {
    try {
      const host = new URL(url).host
      return host.replace(`${props.invitation.subdomain}.`, '')
    } catch {
      return ''
    }
  }
  return ''
})

const suggestion = computed(() => {
  const c = props.invitation.content
  const g = c?.groom?.nickname || c?.groom?.full_name?.split(/[\s,]/)[0] || ''
  const b = c?.bride?.nickname || c?.bride?.full_name?.split(/[\s,]/)[0] || ''
  if (!g || !b) return ''
  const first = c.couple_order === 'bride_first' ? [b, g] : [g, b]
  return sanitizeSubdomain(first.join('-'))
})

function onInput(e: Event) {
  const el = e.target as HTMLInputElement
  const clean = sanitizeSubdomain(el.value)
  if (clean !== el.value) el.value = clean
  setValue(clean)
}

function setValue(v: string) {
  value.value = v
  if (timer) clearTimeout(timer)
  reason.value = ''
  if (v && v === props.invitation.subdomain) {
    state.value = 'same'
    return
  }
  const err = validateSubdomain(v)
  if (!v) {
    state.value = 'idle'
    return
  }
  if (err) {
    state.value = 'invalid'
    reason.value = err
    return
  }
  state.value = 'checking'
  timer = setTimeout(() => check(v), 400)
}

async function check(name: string) {
  const my = ++reqSeq
  try {
    const res = await api.get<{ name: string; available: boolean; reason?: string }>('/subdomains/check', { name })
    if (my !== reqSeq || name !== value.value) return
    state.value = res.available ? 'available' : 'taken'
    reason.value = res.reason ?? ''
  } catch (e) {
    if (my !== reqSeq) return
    state.value = 'error'
    reason.value = errorMessage(e)
  }
}

async function save() {
  if (state.value !== 'available' || saving.value) return
  saving.value = true
  try {
    const inv = await api.put<Invitation>(`/invitations/${props.invitation.id}/subdomain`, { name: value.value })
    toast.success('Subdomain disimpan')
    emit('saved', inv)
  } catch (e) {
    if (e instanceof ApiError && (e.status === 409 || e.fields.name)) {
      state.value = 'taken'
      reason.value = e.fields.name ?? e.message
    } else {
      toast.error(e)
    }
  } finally {
    saving.value = false
  }
}

if (value.value) state.value = 'same'
onBeforeUnmount(() => timer && clearTimeout(timer))
</script>

<template>
  <div class="space-y-2">
    <template v-if="editable">
      <form class="flex flex-col gap-2 sm:flex-row" @submit.prevent="save">
        <div
          class="flex min-w-0 flex-1 items-center overflow-hidden rounded-xl border bg-surface-2 transition focus-within:border-accent focus-within:bg-surface focus-within:ring-3 focus-within:ring-accent/15"
          :class="state === 'invalid' || state === 'taken' ? 'border-danger' : state === 'available' ? 'border-success' : 'border-line'"
        >
          <span class="pl-3 text-sm text-muted">https://</span>
          <input
            :value="value"
            type="text"
            inputmode="url"
            autocapitalize="off"
            autocomplete="off"
            spellcheck="false"
            maxlength="63"
            placeholder="nama-pasangan"
            class="h-10 min-w-0 flex-1 bg-transparent px-0.5 text-sm font-medium text-ink outline-none placeholder:font-normal placeholder:text-muted/70"
            @input="onInput"
          />
          <span v-if="suffix" class="truncate pr-3 text-sm text-muted">.{{ suffix }}</span>
        </div>
        <UiButton type="submit" :disabled="state !== 'available'" :loading="saving">
          <UiIcon name="save" :size="16" /> Simpan
        </UiButton>
      </form>

      <div class="flex min-h-5 items-center gap-1.5 text-xs">
        <template v-if="state === 'checking'">
          <UiSpinner :size="12" class="text-muted" /><span class="text-muted">Mengecek ketersediaan…</span>
        </template>
        <template v-else-if="state === 'available'">
          <UiIcon name="check-circle" :size="14" class="text-success" /><span class="text-success">Tersedia! Klik Simpan untuk memakai.</span>
        </template>
        <template v-else-if="state === 'taken'">
          <UiIcon name="x-circle" :size="14" class="text-danger" /><span class="text-danger">{{ reason || 'Sudah dipakai, coba nama lain.' }}</span>
        </template>
        <template v-else-if="state === 'invalid'">
          <UiIcon name="alert" :size="14" class="text-danger" /><span class="text-danger">{{ reason }}</span>
        </template>
        <template v-else-if="state === 'error'">
          <UiIcon name="alert" :size="14" class="text-warning" /><span class="text-muted">{{ reason }}</span>
        </template>
        <template v-else-if="state === 'same'">
          <UiIcon name="check" :size="14" class="text-success" /><span class="text-muted">Subdomain saat ini.</span>
        </template>
        <template v-else>
          <span class="text-muted">Huruf kecil, angka, dan tanda -, minimal 3 karakter.</span>
          <button v-if="suggestion" type="button" class="ml-1 font-medium text-accent underline" @click="setValue(suggestion)">
            Pakai “{{ suggestion }}”
          </button>
        </template>
      </div>
    </template>

    <template v-else>
      <div class="flex items-center gap-2 rounded-xl border border-line bg-surface-2 px-3 py-2.5 text-sm">
        <UiIcon name="lock" :size="15" class="text-muted" />
        <span v-if="invitation.subdomain" class="min-w-0 truncate font-medium text-ink">
          {{ invitation.subdomain }}<span v-if="suffix" class="font-normal text-muted">.{{ suffix }}</span>
        </span>
        <span v-else class="text-muted">Belum diatur</span>
      </div>
      <p class="text-xs text-muted">{{ lockedHint }}</p>
    </template>
  </div>
</template>
