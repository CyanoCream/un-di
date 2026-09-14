<script setup lang="ts">
import { api, toast, type EventType, type Invitation } from '@undangan/shared'
import UiButton from '@undangan/shared/components/ui/UiButton.vue'
import UiDialog from '@undangan/shared/components/ui/UiDialog.vue'
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const open = defineModel<boolean>('open', { default: false })
const router = useRouter()

const type = ref<EventType>('pernikahan')
const creating = ref(false)

const OPTIONS: { value: EventType; title: string; desc: string }[] = [
  { value: 'pernikahan', title: 'Pernikahan', desc: 'Akad / pemberkatan dan resepsi pernikahan.' },
  { value: 'ngunduh_mantu', title: 'Ngunduh Mantu', desc: 'Acara syukuran di keluarga mempelai pria.' },
]

async function create() {
  creating.value = true
  try {
    const inv = await api.post<Invitation>('/invitations', { event_type: type.value })
    toast.success('Undangan dibuat. Yuk, isi datanya!')
    open.value = false
    router.push(`/undangan/${inv.id}/data`)
  } catch (e) {
    toast.error(e)
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <UiDialog v-model:open="open" title="Buat undangan baru" size="sm" :persistent="creating">
    <p class="mb-4 text-sm text-muted">Pilih jenis acara. Data, tema, dan subdomain bisa diatur setelahnya.</p>
    <div class="space-y-2.5">
      <label
        v-for="o in OPTIONS"
        :key="o.value"
        class="flex cursor-pointer items-start gap-3 rounded-2xl border p-4 transition"
        :class="type === o.value ? 'border-accent bg-accent/6 ring-2 ring-accent/15' : 'border-line hover:bg-surface-2'"
      >
        <input v-model="type" type="radio" name="event-type" :value="o.value" class="sr-only" />
        <span
          class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border-2"
          :class="type === o.value ? 'border-accent bg-accent text-accent-ink' : 'border-line'"
        >
          <UiIcon v-if="type === o.value" name="check" :size="12" :stroke-width="3" />
        </span>
        <span>
          <span class="block font-semibold text-ink">{{ o.title }}</span>
          <span class="block text-sm text-muted">{{ o.desc }}</span>
        </span>
      </label>
    </div>
    <template #footer>
      <UiButton variant="ghost" :disabled="creating" @click="open = false">Batal</UiButton>
      <UiButton :loading="creating" @click="create">Buat undangan</UiButton>
    </template>
  </UiDialog>
</template>
