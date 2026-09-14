<script setup lang="ts">
// Komponen internal InvitationEditor: mengubah objek `person` milik state lokal editor secara langsung.
import { useId } from 'vue'
import type { Person } from '../../types'
import ImageUpload from '../ImageUpload.vue'
import UiField from '../ui/UiField.vue'
import UiInput from '../ui/UiInput.vue'

const props = defineProps<{ person: Person; gender: 'pria' | 'wanita' }>()

const listId = `child-order-${useId()}`
const prefix = props.gender === 'pria' ? 'Putra' : 'Putri'
const suggestions = ['pertama', 'kedua', 'ketiga', 'keempat', 'kelima', 'bungsu', 'tunggal'].map((s) => `${prefix} ${s}`)

function setInstagram(v: string) {
  props.person.instagram = v
    .trim()
    .replace(/^https?:\/\/(www\.)?instagram\.com\//i, '')
    .replace(/^@+/, '')
    .replace(/[/?#].*$/, '')
}
</script>

<template>
  <div class="space-y-4">
    <div class="grid gap-4 sm:grid-cols-2">
      <UiField label="Nama lengkap" hint="Boleh dengan gelar, mis. “Hardiansyah, S.Kom.”" class="sm:col-span-2">
        <UiInput v-model="person.full_name" :placeholder="gender === 'pria' ? 'Nama lengkap mempelai pria' : 'Nama lengkap mempelai wanita'" />
      </UiField>
      <UiField label="Nama panggilan">
        <UiInput v-model="person.nickname" :placeholder="gender === 'pria' ? 'mis. Raka' : 'mis. Nadia'" />
      </UiField>
      <UiField label="Anak ke-" hint="Pilih saran atau ketik sendiri">
        <UiInput v-model="person.child_order" :list="listId" :placeholder="`${prefix} pertama`" />
        <datalist :id="listId">
          <option v-for="s in suggestions" :key="s" :value="s" />
        </datalist>
      </UiField>
    </div>

    <UiField label="Foto" hint="Opsional. Tema tanpa foto mempelai akan menyembunyikannya.">
      <ImageUpload v-model="person.photo" aspect="3 / 4" />
    </UiField>

    <div class="grid gap-4 sm:grid-cols-2">
      <div class="space-y-2">
        <UiField label="Nama ayah">
          <UiInput v-model="person.father" placeholder="mis. Bapak Suyadi" />
        </UiField>
        <label class="flex items-center gap-2 text-sm text-muted">
          <input v-model="person.father_deceased" type="checkbox" class="size-4 rounded accent-accent" />
          Sudah almarhum <span class="text-xs">(tampil “Alm.”)</span>
        </label>
      </div>
      <div class="space-y-2">
        <UiField label="Nama ibu">
          <UiInput v-model="person.mother" placeholder="mis. Ibu Sulastri" />
        </UiField>
        <label class="flex items-center gap-2 text-sm text-muted">
          <input v-model="person.mother_deceased" type="checkbox" class="size-4 rounded accent-accent" />
          Sudah almarhumah <span class="text-xs">(tampil “Almh.”)</span>
        </label>
      </div>
    </div>

    <UiField label="Instagram" hint="Tanpa @, opsional">
      <div class="flex items-center">
        <span class="flex h-10 items-center rounded-l-xl border border-r-0 border-line bg-surface px-3 text-sm text-muted">@</span>
        <UiInput
          :model-value="person.instagram"
          class="rounded-l-none!"
          placeholder="username"
          autocapitalize="off"
          @update:model-value="setInstagram(String($event))"
        />
      </div>
    </UiField>
  </div>
</template>
