import { setUnauthorizedHandler } from '@undangan/shared'
import { createRouter, createWebHistory } from 'vue-router'
import { auth, fetchMe, logout } from './lib/auth'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (_to, _from, saved) => saved ?? { top: 0 },
  routes: [
    { path: '/login', name: 'login', component: () => import('./pages/LoginPage.vue'), meta: { public: true, title: 'Masuk' } },
    {
      path: '/',
      component: () => import('./components/AppLayout.vue'),
      children: [
        { path: '', name: 'dashboard', component: () => import('./pages/DashboardPage.vue'), meta: { title: 'Dashboard' } },
        { path: 'orders', name: 'orders', component: () => import('./pages/OrdersPage.vue'), meta: { title: 'Order Pembayaran' } },
        { path: 'users', name: 'users', component: () => import('./pages/UsersPage.vue'), meta: { title: 'Pengguna' } },
        { path: 'users/:id', name: 'user-detail', component: () => import('./pages/UserDetailPage.vue'), meta: { title: 'Detail Pengguna' } },
        { path: 'invitations', name: 'invitations', component: () => import('./pages/InvitationsPage.vue'), meta: { title: 'Undangan' } },
        {
          path: 'invitations/:id',
          name: 'invitation-detail',
          component: () => import('./pages/InvitationDetailPage.vue'),
          meta: { title: 'Detail Undangan' },
        },
        { path: 'subscriptions', name: 'subscriptions', component: () => import('./pages/SubscriptionsPage.vue'), meta: { title: 'Langganan' } },
        { path: 'plans', name: 'plans', component: () => import('./pages/PlansPage.vue'), meta: { title: 'Paket' } },
        { path: 'themes', name: 'themes', component: () => import('./pages/ThemesPage.vue'), meta: { title: 'Tema' } },
        { path: 'music', name: 'music', component: () => import('./pages/MusicPage.vue'), meta: { title: 'Pustaka Musik' } },
        { path: 'settings', name: 'settings', component: () => import('./pages/SettingsPage.vue'), meta: { title: 'Pengaturan Pembayaran' } },
        { path: 'audit', name: 'audit', component: () => import('./pages/AuditPage.vue'), meta: { title: 'Audit Log' } },
        { path: ':pathMatch(.*)*', name: 'not-found', component: () => import('./pages/NotFoundPage.vue'), meta: { title: 'Tidak ditemukan' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (!auth.checked) await fetchMe()

  if (auth.user && auth.user.role !== 'super_admin') {
    await logout()
    return { name: 'login', query: { error: 'role' } }
  }
  if (to.meta.public) return auth.user ? { name: 'dashboard' } : true
  if (!auth.user) return { name: 'login', query: to.fullPath !== '/' ? { redirect: to.fullPath } : {} }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · Admin Console` : 'Admin Console'
})

setUnauthorizedHandler(() => {
  auth.user = null
  const current = router.currentRoute.value
  if (current.name !== 'login') {
    router.push({ name: 'login', query: { redirect: current.fullPath, error: 'session' } })
  }
})
