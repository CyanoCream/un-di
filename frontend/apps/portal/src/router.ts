import { toast } from '@undangan/shared'
import { createRouter, createWebHistory } from 'vue-router'
import { auth, ensureSession, logout } from './lib/auth'
import { APP_NAME } from './lib/brand'
import { savePendingTheme } from './lib/pendingTheme'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    /** Hanya untuk pengunjung yang belum login. */
    guest?: boolean
    /** Sembunyikan tab bawah (halaman fokus seperti editor). */
    focus?: boolean
    /** Terbuka untuk siapa saja (tanpa cek sesi), mis. stasiun check-in. */
    public?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  scrollBehavior(to, from, saved) {
    if (saved) return saved
    // Pindah tab di dalam editor undangan tidak perlu scroll ke atas.
    if (to.params.id && to.params.id === from.params.id && to.path.startsWith('/undangan/')) return false
    return { top: 0 }
  },
  routes: [
    { path: '/masuk', name: 'login', component: () => import('./pages/LoginPage.vue'), meta: { guest: true, title: 'Masuk' } },
    { path: '/daftar', name: 'register', component: () => import('./pages/RegisterPage.vue'), meta: { guest: true, title: 'Daftar' } },
    // Stasiun penerima tamu: publik (PIN), tanpa layout aplikasi.
    { path: '/checkin/:id', name: 'checkin', component: () => import('./pages/CheckinPage.vue'), props: true, meta: { public: true, title: 'Check-in Tamu' } },
    {
      path: '/',
      component: () => import('./layouts/AppLayout.vue'),
      children: [
        { path: '', name: 'home', component: () => import('./pages/HomePage.vue'), meta: { title: 'Beranda' } },
        { path: 'paket', name: 'plans', component: () => import('./pages/PlansPage.vue'), meta: { title: 'Paket' } },
        { path: 'pembayaran', name: 'orders', component: () => import('./pages/OrdersPage.vue'), meta: { title: 'Pembayaran' } },
        {
          path: 'pembayaran/:id',
          name: 'order',
          component: () => import('./pages/OrderDetailPage.vue'),
          props: true,
          meta: { title: 'Pembayaran' },
        },
        { path: 'undangan', name: 'invitations', component: () => import('./pages/InvitationsPage.vue'), meta: { title: 'Undangan' } },
        {
          path: 'undangan/:id',
          component: () => import('./pages/invitation/InvitationLayout.vue'),
          props: true,
          meta: { focus: true },
          children: [
            { path: '', name: 'invitation', redirect: (to) => ({ name: 'inv-data', params: to.params }) },
            { path: 'data', name: 'inv-data', component: () => import('./pages/invitation/DataTab.vue'), meta: { title: 'Isi Data' } },
            { path: 'tema', name: 'inv-theme', component: () => import('./pages/invitation/ThemeTab.vue'), meta: { title: 'Tema' } },
            { path: 'tamu', name: 'inv-guests', component: () => import('./pages/invitation/GuestsTab.vue'), meta: { title: 'Tamu' } },
            { path: 'kehadiran', name: 'inv-attendance', component: () => import('./pages/invitation/AttendanceTab.vue'), meta: { title: 'Kehadiran' } },
            { path: 'ucapan', name: 'inv-wishes', component: () => import('./pages/invitation/WishesTab.vue'), meta: { title: 'Ucapan' } },
            { path: 'hadiah', name: 'inv-gifts', component: () => import('./pages/invitation/GiftsTab.vue'), meta: { title: 'Hadiah' } },
            { path: 'bagikan', name: 'inv-share', component: () => import('./pages/invitation/ShareTab.vue'), meta: { title: 'Bagikan' } },
          ],
        },
        { path: 'akun', name: 'account', component: () => import('./pages/AccountPage.vue'), meta: { title: 'Akun' } },
      ],
    },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('./pages/NotFoundPage.vue'), meta: { title: 'Tidak ditemukan' } },
  ],
})

router.beforeEach(async (to) => {
  // Tema pilihan dari landing page (/daftar?tema=<slug>) — ingat untuk disorot di tab Tema.
  if (to.query.tema) savePendingTheme(to.query.tema)
  if (to.meta.public) return true
  await ensureSession()
  const user = auth.user

  if (user && user.role !== 'customer') {
    await logout()
    toast.error('Akun ini bukan akun customer. Silakan masuk melalui portal admin.')
    return to.meta.guest ? true : { name: 'login' }
  }

  if (to.meta.guest) {
    if (user) {
      const next = typeof to.query.next === 'string' && to.query.next.startsWith('/') ? to.query.next : '/'
      return next
    }
    return true
  }

  if (to.name === 'not-found') return true

  if (!user) {
    return { name: 'login', query: to.fullPath !== '/' ? { next: to.fullPath } : {} }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · ${APP_NAME}` : `Portal ${APP_NAME}`
})

// Chunk lama setelah deploy → muat ulang halaman.
router.onError((err, to) => {
  if (/Failed to fetch dynamically imported module|Importing a module script failed/i.test(String(err?.message))) {
    window.location.assign(to.fullPath)
  }
})
