<script setup lang="ts">
import { ref, watch } from 'vue'
import UiSpinner from './ui/UiSpinner.vue'

const props = withDefaults(defineProps<{ src: string; reloadKey?: number; device?: 'mobile' | 'desktop' }>(), {
  reloadKey: 0,
  device: 'mobile',
})

const frame = ref<HTMLIFrameElement | null>(null)
/** initial = belum pernah tampil (overlay penuh); reload = muat ulang (indikator kecil). */
const state = ref<'initial' | 'reload' | 'ready'>('initial')
let savedScroll = 0

function onLoad() {
  if (savedScroll) {
    try {
      frame.value?.contentWindow?.scrollTo(0, savedScroll)
    } catch {
      /* lintas origin — abaikan */
    }
  }
  savedScroll = 0
  state.value = 'ready'
}

watch(
  () => props.src,
  () => {
    savedScroll = 0
    state.value = 'initial'
  },
)

watch(
  () => props.reloadKey,
  () => {
    const f = frame.value
    if (!f) return
    state.value = state.value === 'initial' ? 'initial' : 'reload'
    try {
      savedScroll = f.contentWindow?.scrollY ?? 0
      f.contentWindow?.location.reload()
    } catch {
      savedScroll = 0
      f.src = props.src
    }
  },
)
</script>

<template>
  <div class="relative flex h-full w-full items-center justify-center">
    <div
      class="relative shrink-0 overflow-hidden bg-surface"
      :class="
        device === 'mobile'
          ? 'h-[760px] max-h-full min-h-[420px] w-[372px] max-w-full rounded-[2.25rem] border-[10px] border-ink shadow-xl'
          : 'h-full min-h-[480px] w-full rounded-xl border border-line shadow-sm'
      "
    >
      <div
        v-if="device === 'mobile'"
        class="absolute top-1.5 left-1/2 z-10 h-4 w-20 -translate-x-1/2 rounded-full bg-ink"
        aria-hidden="true"
      />
      <iframe
        ref="frame"
        :src="src"
        title="Pratinjau undangan"
        class="block h-full w-full border-0 bg-surface"
        @load="onLoad"
      />
      <div v-if="state === 'initial'" class="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-surface text-muted">
        <UiSpinner :size="24" />
        <span class="text-xs">Memuat pratinjau…</span>
      </div>
      <div
        v-else-if="state === 'reload'"
        class="absolute top-6 right-3 z-10 flex items-center gap-1.5 rounded-full border border-line bg-surface/95 px-2.5 py-1 text-xs text-muted shadow"
      >
        <UiSpinner :size="12" /> Memperbarui
      </div>
    </div>
  </div>
</template>
