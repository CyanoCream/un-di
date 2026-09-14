<script setup lang="ts">
import { ref } from 'vue'
import { copyText } from '../lib/util'
import AppIcon from './AppIcon.vue'

const props = withDefaults(defineProps<{ text: string; label?: string; small?: boolean }>(), { small: true })
const copied = ref(false)

async function copy() {
  if (await copyText(props.text)) {
    copied.value = true
    window.setTimeout(() => (copied.value = false), 1500)
  }
}
</script>

<template>
  <button
    type="button"
    class="btn btn-ghost"
    :class="[small && 'btn-sm', !label && 'btn-icon', copied && 'text-success hover:text-success']"
    :aria-label="label ? undefined : 'Salin'"
    :title="copied ? 'Tersalin' : 'Salin'"
    @click.stop="copy"
  >
    <AppIcon :name="copied ? 'check' : 'copy'" :size="14" />
    <span v-if="label">{{ copied ? 'Tersalin' : label }}</span>
  </button>
</template>
