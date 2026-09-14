<script setup lang="ts">
import { formatRupiah } from '@undangan/shared'
import { computed } from 'vue'

/** Nominal order dengan 3 digit terakhir (kode unik) disorot supaya mudah dicocokkan di mutasi. */
const props = defineProps<{ amount: number; uniqueCode?: number | null }>()

const parts = computed(() => {
  const text = formatRupiah(props.amount)
  const code = props.uniqueCode ?? 0
  // Sorot grup ribuan terakhir jika kode unik < 1000 (harga paket kelipatan ribuan).
  const idx = text.lastIndexOf('.')
  if (!code || code >= 1000 || idx < 0 || (props.amount - code) % 1000 !== 0) return { head: text, tail: '' }
  return { head: text.slice(0, idx + 1), tail: text.slice(idx + 1) }
})
</script>

<template>
  <span class="num whitespace-nowrap">
    {{ parts.head }}<mark v-if="parts.tail" class="rounded-sm bg-accent/15 px-0.5 text-accent" :title="`Kode unik ${uniqueCode}`">{{ parts.tail }}</mark>
  </span>
</template>
