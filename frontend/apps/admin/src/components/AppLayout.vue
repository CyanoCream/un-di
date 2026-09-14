<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { auth, logout } from '../lib/auth'
import { pendingOrders, refreshPendingOrders } from '../lib/pending'
import { initials } from '../lib/util'
import AppIcon, { type IconName } from './AppIcon.vue'

interface NavItem {
  to: string
  label: string
  icon: IconName
  badge?: 'orders'
}

const NAV: { title: string; items: NavItem[] }[] = [
  { title: 'Ringkasan', items: [{ to: '/', label: 'Dashboard', icon: 'dashboard' }] },
  {
    title: 'Operasional',
    items: [
      { to: '/orders', label: 'Order', icon: 'orders', badge: 'orders' },
      { to: '/subscriptions', label: 'Langganan', icon: 'card' },
      { to: '/invitations', label: 'Undangan', icon: 'mail' },
      { to: '/users', label: 'Pengguna', icon: 'users' },
    ],
  },
  {
    title: 'Katalog',
    items: [
      { to: '/plans', label: 'Paket', icon: 'layers' },
      { to: '/themes', label: 'Tema', icon: 'palette' },
      { to: '/music', label: 'Musik', icon: 'music' },
    ],
  },
  {
    title: 'Sistem',
    items: [
      { to: '/settings', label: 'Pengaturan', icon: 'settings' },
      { to: '/audit', label: 'Audit Log', icon: 'audit' },
    ],
  },
]

const route = useRoute()
const router = useRouter()

const collapsed = ref(localStorage.getItem('admin.sidebar') === 'collapsed')
const mobileOpen = ref(false)
const menuOpen = ref(false)
const search = ref('')
const menuRef = ref<HTMLElement | null>(null)

watch(collapsed, (v) => localStorage.setItem('admin.sidebar', v ? 'collapsed' : 'expanded'))
watch(
  () => route.fullPath,
  () => {
    mobileOpen.value = false
    menuOpen.value = false
  },
)

function isActive(to: string) {
  return to === '/' ? route.path === '/' : route.path === to || route.path.startsWith(to + '/')
}

function submitSearch() {
  const q = search.value.trim()
  router.push({ path: '/users', query: q ? { q } : {} })
  ;(document.activeElement as HTMLElement | null)?.blur()
}

async function doLogout() {
  await logout()
  router.push({ name: 'login' })
}

function onDocClick(e: MouseEvent) {
  if (menuOpen.value && menuRef.value && !menuRef.value.contains(e.target as Node)) menuOpen.value = false
}
function onDocKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    menuOpen.value = false
    mobileOpen.value = false
  }
  const tag = (e.target as HTMLElement | null)?.tagName
  if (e.key === '/' && tag !== 'INPUT' && tag !== 'TEXTAREA' && tag !== 'SELECT' && !(e.target as HTMLElement)?.isContentEditable) {
    e.preventDefault()
    document.getElementById('global-search')?.focus()
  }
}

let timer: number | undefined
onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onDocKey)
  refreshPendingOrders()
  timer = window.setInterval(refreshPendingOrders, 60_000)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocKey)
  window.clearInterval(timer)
})
</script>

<template>
  <div class="min-h-dvh">
    <a href="#main" class="sr-only z-[80] rounded bg-accent px-3 py-2 text-accent-ink focus:not-sr-only focus:fixed focus:top-2 focus:left-2">Lewati ke konten</a>

    <!-- Overlay mobile -->
    <Transition enter-active-class="transition-opacity" enter-from-class="opacity-0" leave-active-class="transition-opacity" leave-to-class="opacity-0">
      <div v-if="mobileOpen" class="fixed inset-0 z-30 bg-black/60 lg:hidden" @click="mobileOpen = false" />
    </Transition>

    <!-- Sidebar -->
    <aside
      class="fixed inset-y-0 left-0 z-30 flex w-60 flex-col border-r border-line bg-surface transition-[width,transform] duration-200 lg:translate-x-0"
      :class="[mobileOpen ? 'translate-x-0' : '-translate-x-full', collapsed ? 'lg:w-16' : 'lg:w-60']"
      aria-label="Navigasi utama"
    >
      <div class="flex h-14 shrink-0 items-center gap-2.5 border-b border-line px-4">
        <span class="grid size-8 shrink-0 place-items-center rounded-md bg-accent/15 text-accent ring-1 ring-accent/30">
          <AppIcon name="mail" :size="16" :stroke="2" />
        </span>
        <div class="min-w-0 leading-tight" :class="collapsed && 'lg:hidden'">
          <p class="truncate text-sm font-semibold">Admin Console</p>
          <p class="truncate text-[11px] text-muted">Undangan Digital</p>
        </div>
        <button type="button" class="btn btn-ghost btn-icon ml-auto h-7 w-7 lg:hidden" aria-label="Tutup menu" @click="mobileOpen = false">
          <AppIcon name="x" />
        </button>
      </div>

      <nav class="flex-1 overflow-y-auto px-2 py-3">
        <div v-for="group in NAV" :key="group.title" class="mb-4">
          <p class="mb-1 px-2.5 text-[10px] font-medium tracking-wider text-muted/70 uppercase" :class="collapsed && 'lg:invisible lg:h-0 lg:mb-0'">
            {{ group.title }}
          </p>
          <RouterLink
            v-for="item in group.items"
            :key="item.to"
            :to="item.to"
            class="group relative mb-0.5 flex h-8 items-center gap-2.5 rounded-md px-2.5 text-[13px] font-medium transition-colors"
            :class="[
              isActive(item.to) ? 'bg-accent/10 text-ink' : 'text-muted hover:bg-surface-2 hover:text-ink',
              collapsed && 'lg:justify-center lg:px-0',
            ]"
            :title="collapsed ? item.label : undefined"
            :aria-current="isActive(item.to) ? 'page' : undefined"
          >
            <span v-if="isActive(item.to)" class="absolute top-1.5 bottom-1.5 left-0 w-0.5 rounded-full bg-accent" />
            <AppIcon :name="item.icon" :size="16" :class="isActive(item.to) ? 'text-accent' : ''" />
            <span class="truncate" :class="collapsed && 'lg:sr-only'">{{ item.label }}</span>
            <span
              v-if="item.badge === 'orders' && pendingOrders"
              class="num ml-auto rounded-full bg-warning/15 px-1.5 text-[11px] font-semibold text-warning"
              :class="collapsed && 'lg:absolute lg:top-0 lg:right-1 lg:ml-0 lg:px-1 lg:text-[9px]'"
            >
              {{ pendingOrders }}
            </span>
          </RouterLink>
        </div>
      </nav>

      <div class="hidden border-t border-line p-2 lg:block">
        <button
          type="button"
          class="btn btn-ghost h-8 w-full justify-start gap-2.5 px-2.5 text-[13px]"
          :class="collapsed && 'justify-center px-0'"
          :aria-label="collapsed ? 'Perlebar sidebar' : 'Ciutkan sidebar'"
          :aria-expanded="!collapsed"
          @click="collapsed = !collapsed"
        >
          <AppIcon :name="collapsed ? 'chevron-right' : 'chevron-left'" />
          <span v-if="!collapsed">Ciutkan</span>
        </button>
      </div>
    </aside>

    <!-- Konten -->
    <div class="flex min-h-dvh flex-col transition-[padding] duration-200" :class="collapsed ? 'lg:pl-16' : 'lg:pl-60'">
      <header class="sticky top-0 z-20 flex h-14 items-center gap-3 border-b border-line bg-canvas/85 px-4 backdrop-blur lg:px-6">
        <button type="button" class="btn btn-ghost btn-icon -ml-2 lg:hidden" aria-label="Buka menu" @click="mobileOpen = true">
          <AppIcon name="menu" :size="18" />
        </button>

        <form role="search" class="relative w-full max-w-md" @submit.prevent="submitSearch">
          <label for="global-search" class="sr-only">Cari pengguna</label>
          <AppIcon name="search" class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted" />
          <input
            id="global-search"
            v-model="search"
            type="search"
            placeholder="Cari pengguna (nama, email, no. HP)…"
            class="input h-8 bg-surface pr-10 pl-9"
            autocomplete="off"
          />
          <span class="kbd pointer-events-none absolute top-1/2 right-2 hidden -translate-y-1/2 sm:inline">/</span>
        </form>

        <div ref="menuRef" class="relative ml-auto">
          <button
            type="button"
            class="flex h-9 items-center gap-2 rounded-md px-1.5 text-left hover:bg-surface-2"
            aria-haspopup="menu"
            :aria-expanded="menuOpen"
            @click="menuOpen = !menuOpen"
          >
            <span class="grid size-7 place-items-center rounded-full bg-accent/20 text-[11px] font-semibold text-accent">
              {{ initials(auth.user?.name) }}
            </span>
            <span class="hidden max-w-40 truncate text-[13px] font-medium sm:block">{{ auth.user?.name }}</span>
            <AppIcon name="chevron-down" :size="14" class="hidden text-muted sm:block" />
          </button>
          <Transition enter-active-class="transition duration-100" enter-from-class="opacity-0 -translate-y-1" leave-active-class="transition duration-75" leave-to-class="opacity-0">
            <div v-if="menuOpen" role="menu" class="absolute right-0 mt-1.5 w-60 overflow-hidden rounded-lg border border-line bg-surface-2 shadow-xl shadow-black/40">
              <div class="border-b border-line px-3 py-2.5">
                <p class="truncate text-sm font-medium">{{ auth.user?.name }}</p>
                <p class="truncate text-xs text-muted">{{ auth.user?.email }}</p>
                <p class="mt-1.5 inline-flex items-center gap-1 text-[11px] text-accent"><AppIcon name="shield" :size="12" /> Super admin</p>
              </div>
              <button type="button" role="menuitem" class="flex w-full items-center gap-2 px-3 py-2 text-sm text-ink hover:bg-line/60" @click="doLogout">
                <AppIcon name="logout" class="text-muted" /> Keluar
              </button>
            </div>
          </Transition>
        </div>
      </header>

      <main id="main" class="mx-auto w-full max-w-[1440px] flex-1 px-4 py-5 lg:px-6">
        <RouterView v-slot="{ Component, route: r }">
          <component :is="Component" :key="r.path" />
        </RouterView>
      </main>
    </div>
  </div>
</template>
