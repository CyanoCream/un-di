import { setUnauthorizedHandler, toast } from '@undangan/shared'
import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { auth, clearSession } from './lib/auth'
import { router } from './router'

setUnauthorizedHandler(() => {
  const wasLoggedIn = !!auth.user
  clearSession()
  const current = router.currentRoute.value
  if (current.meta.guest) return
  if (wasLoggedIn) toast.info('Sesi Anda telah berakhir. Silakan masuk kembali.')
  router.push({ name: 'login', query: { next: current.fullPath } })
})

createApp(App).use(router).mount('#app')
