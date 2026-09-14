<script setup lang="ts">
import UiIcon from '@undangan/shared/components/ui/UiIcon.vue'
import { LANDING_URL } from '../lib/brand'
import BrandMark from './BrandMark.vue'

defineProps<{ title: string; subtitle?: string }>()
</script>

<template>
  <div class="relative flex min-h-dvh flex-col overflow-hidden bg-page">
    <!-- Ornamen bunga garis -->
    <svg
      class="pointer-events-none absolute -top-10 -right-16 w-72 text-accent/25 sm:w-96"
      viewBox="0 0 300 300"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      aria-hidden="true"
    >
      <path d="M150 150c-30-40-20-90 10-110 25 30 25 75-10 110Z" />
      <path d="M150 150c40-30 90-20 110 10-30 25-75 25-110-10Z" />
      <path d="M150 150c30 40 20 90-10 110-25-30-25-75 10-110Z" />
      <path d="M150 150c-40 30-90 20-110-10 30-25 75-25 110 10Z" />
      <circle cx="150" cy="150" r="9" />
      <path d="M40 40c40 10 70 40 90 90" />
      <path d="M60 20c-5 20 5 35 25 40M20 70c20-5 35 5 40 25" />
      <path d="M260 260c-40-10-70-40-90-90" />
      <path d="M240 280c5-20-5-35-25-40M280 230c-20 5-35-5-40-25" />
    </svg>
    <svg
      class="pointer-events-none absolute -bottom-20 -left-20 w-80 rotate-45 text-accent/20 sm:w-[28rem]"
      viewBox="0 0 300 300"
      fill="none"
      stroke="currentColor"
      stroke-width="1.2"
      aria-hidden="true"
    >
      <path d="M20 280C80 220 110 160 120 60" />
      <path d="M120 60c-25 5-40 25-35 50 25-5 40-25 35-50Z" />
      <path d="M105 130c-25-5-45 5-50 30 25 5 45-5 50-30Z" />
      <path d="M112 110c25-10 45-2 55 22-24 10-45 2-55-22Z" />
      <path d="M85 190c-25 0-42 14-45 38 25 0 42-14 45-38Z" />
      <path d="M95 175c22-14 45-10 60 10-22 14-45 10-60-10Z" />
      <circle cx="120" cy="52" r="4" />
    </svg>

    <div class="relative z-10 px-4 pt-4 sm:px-6 sm:pt-6">
      <a
        :href="LANDING_URL"
        class="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium text-muted transition hover:bg-surface/80 hover:text-ink"
      >
        <UiIcon name="arrow-left" :size="16" /> Kembali ke beranda
      </a>
    </div>

    <div class="relative z-10 flex flex-1 flex-col items-center justify-center px-4 pt-4 pb-10">
      <a :href="LANDING_URL" class="mb-8" aria-label="Beranda"><BrandMark /></a>
      <div class="w-full max-w-md rounded-3xl border border-line bg-surface/90 p-6 shadow-lift backdrop-blur sm:p-8">
        <div class="mb-6 text-center">
          <div class="mx-auto mb-3 flex items-center justify-center gap-3 text-accent/60" aria-hidden="true">
            <span class="h-px w-10 bg-current" />
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2l2.4 7.6L22 12l-7.6 2.4L12 22l-2.4-7.6L2 12l7.6-2.4Z" /></svg>
            <span class="h-px w-10 bg-current" />
          </div>
          <h1 class="heading text-3xl sm:text-4xl">{{ title }}</h1>
          <p v-if="subtitle" class="mt-2 text-sm text-muted">{{ subtitle }}</p>
        </div>
        <slot />
      </div>
      <div class="mt-6 text-center text-sm text-muted">
        <slot name="footer" />
      </div>
    </div>
  </div>
</template>
