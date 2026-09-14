import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

const backend = 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    port: 5174,
    proxy: {
      '/api': { target: backend, changeOrigin: false },
      '/uploads': { target: backend, changeOrigin: false },
      '/_preview': { target: backend, changeOrigin: false },
      '/_theme': { target: backend, changeOrigin: false },
      '/_shared': { target: backend, changeOrigin: false },
    },
  },
})
