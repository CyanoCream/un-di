<script setup lang="ts">
import UiIcon, { type IconName } from '@undangan/shared/components/ui/UiIcon.vue'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import BrandMark from '../components/BrandMark.vue'
import { auth } from '../lib/auth'

const route = useRoute()

const tabs: { to: string; label: string; icon: IconName; match: (p: string) => boolean }[] = [
  { to: '/', label: 'Beranda', icon: 'home', match: (p) => p === '/' },
  { to: '/undangan', label: 'Undangan', icon: 'mail', match: (p) => p.startsWith('/undangan') },
  { to: '/paket', label: 'Paket', icon: 'gift', match: (p) => p.startsWith('/paket') || p.startsWith('/pembayaran') },
  { to: '/akun', label: 'Akun', icon: 'user', match: (p) => p.startsWith('/akun') },
]

const focus = computed(() => route.matched.some((r) => r.meta.focus))
const initials = computed(() =>
  (auth.user?.name ?? '?')
    .split(/\s+/)
    .slice(0, 2)
    .map((s) => s.charAt(0))
    .join('')
    .toUpperCase(),
)
</script>

<template>
  <div class="min-h-dvh bg-page">
    <!-- Top bar (desktop) -->
    <header class="sticky top-0 z-40 hidden border-b border-line/80 bg-page/85 backdrop-blur md:block">
      <div class="mx-auto flex h-16 max-w-7xl items-center gap-8 px-6">
        <RouterLink to="/"><BrandMark size="sm" /></RouterLink>
        <nav class="flex items-center gap-1">
          <RouterLink
            v-for="t in tabs"
            :key="t.to"
            :to="t.to"
            class="rounded-full px-4 py-2 text-sm font-medium transition"
            :class="t.match(route.path) ? 'bg-accent/10 text-accent' : 'text-muted hover:bg-surface-2 hover:text-ink'"
          >
            {{ t.label }}
          </RouterLink>
        </nav>
        <RouterLink to="/akun" class="ml-auto flex items-center gap-2.5 rounded-full py-1 pr-1 pl-3 transition hover:bg-surface-2">
          <span class="max-w-40 truncate text-sm text-ink">{{ auth.user?.name }}</span>
          <span class="flex size-8 items-center justify-center rounded-full bg-accent-soft text-xs font-semibold text-accent">{{ initials }}</span>
        </RouterLink>
      </div>
    </header>

    <!-- Header mobile -->
    <header v-if="!focus" class="flex items-center justify-between px-4 pt-[max(1rem,env(safe-area-inset-top))] pb-2 md:hidden">
      <RouterLink to="/"><BrandMark size="sm" /></RouterLink>
      <RouterLink to="/akun" class="flex size-9 items-center justify-center rounded-full bg-accent-soft text-xs font-semibold text-accent">
        {{ initials }}
      </RouterLink>
    </header>

    <main :class="focus ? 'pb-4' : 'pb-28 md:pb-12'">
      <RouterView />
    </main>

    <!-- Tab bawah (mobile) -->
    <nav
      v-if="!focus"
      class="fixed inset-x-0 bottom-0 z-40 border-t border-line bg-surface/95 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
      aria-label="Navigasi utama"
    >
      <div class="mx-auto grid max-w-md grid-cols-4">
        <RouterLink
          v-for="t in tabs"
          :key="t.to"
          :to="t.to"
          class="flex flex-col items-center gap-1 py-2 text-[11px] font-medium transition"
          :class="t.match(route.path) ? 'text-accent' : 'text-muted'"
        >
          <span class="flex h-7 w-12 items-center justify-center rounded-full transition" :class="t.match(route.path) ? 'bg-accent/12' : ''">
            <UiIcon :name="t.icon" :size="20" />
          </span>
          {{ t.label }}
        </RouterLink>
      </div>
    </nav>
  </div>
</template>
