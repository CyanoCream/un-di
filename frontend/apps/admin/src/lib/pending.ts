import { api, type Order, type Paginated } from '@undangan/shared'
import { ref } from 'vue'

/** Jumlah order menunggu konfirmasi — dipakai badge sidebar. */
export const pendingOrders = ref<number | null>(null)

export async function refreshPendingOrders() {
  try {
    const res = await api.get<Paginated<Order>>('/admin/orders', { status: 'awaiting_confirmation', per_page: 1 })
    pendingOrders.value = res.total
  } catch {
    /* badge tidak kritis */
  }
}
