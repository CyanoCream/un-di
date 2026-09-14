import { api, type InvitationSummary, type Paginated, type Subscription } from '@undangan/shared'
import { computed, ref } from 'vue'

/** Status langganan + jumlah undangan untuk menentukan boleh membuat undangan. */
export function useAccountState() {
  const subscription = ref<Subscription | null>(null)
  const history = ref<Subscription[]>([])
  const invitations = ref<InvitationSummary[]>([])
  const invitationTotal = ref(0)
  const loading = ref(true)
  const error = ref<unknown>(null)

  async function load() {
    loading.value = true
    error.value = null
    try {
      const [sub, inv] = await Promise.all([
        api.get<{ current: Subscription | null; history: Subscription[] }>('/me/subscription'),
        api.get<Paginated<InvitationSummary>>('/invitations', { per_page: 50 }),
      ])
      subscription.value = sub.current
      history.value = sub.history ?? []
      invitations.value = inv.items ?? []
      invitationTotal.value = inv.total ?? invitations.value.length
    } catch (e) {
      error.value = e
    } finally {
      loading.value = false
    }
  }

  const createBlockReason = computed<string | null>(() => {
    const s = subscription.value
    if (!s || s.status === 'expired' || s.status === 'cancelled') return 'Anda belum memiliki paket aktif. Pilih paket terlebih dahulu untuk membuat undangan.'
    if (s.status === 'grace') return 'Masa aktif paket Anda sudah habis. Perpanjang paket untuk membuat undangan baru.'
    if (s.max_invitations > 0 && invitationTotal.value >= s.max_invitations)
      return `Kuota undangan paket Anda sudah terpakai (${invitationTotal.value}/${s.max_invitations}). Upgrade paket untuk menambah undangan.`
    return null
  })

  return { subscription, history, invitations, invitationTotal, loading, error, load, createBlockReason }
}
