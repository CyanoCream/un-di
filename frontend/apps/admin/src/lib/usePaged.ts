import { api, type Paginated } from '@undangan/shared'
import { onMounted, ref, watch, type Ref } from 'vue'
import { errMsg } from './util'

type QueryValue = string | number | boolean | undefined | null

/** List ber-halaman dengan filter reaktif. Mengubah filter → kembali ke halaman 1. */
export function usePaged<T>(path: string, query: () => Record<string, QueryValue> = () => ({}), perPage = 20) {
  const items = ref([]) as Ref<T[]>
  const total = ref(0)
  const page = ref(1)
  const loading = ref(true)
  const error = ref('')
  let seq = 0

  async function load() {
    const id = ++seq
    loading.value = true
    error.value = ''
    try {
      const res = await api.get<Paginated<T>>(path, { ...query(), page: page.value, per_page: perPage })
      if (id !== seq) return
      items.value = res.items ?? []
      total.value = res.total ?? 0
    } catch (e) {
      if (id === seq) error.value = errMsg(e)
    } finally {
      if (id === seq) loading.value = false
    }
  }

  watch(query, () => (page.value === 1 ? load() : (page.value = 1)), { deep: true })
  watch(page, load)
  onMounted(load)

  return { items, total, page, perPage, loading, error, load }
}
