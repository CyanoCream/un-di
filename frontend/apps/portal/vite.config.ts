import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

const API = 'http://localhost:8080'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': { target: API, changeOrigin: false },
      '/uploads': { target: API, changeOrigin: false },
      '/_preview': { target: API, changeOrigin: false },
      '/_theme': { target: API, changeOrigin: false },
      '/_shared': { target: API, changeOrigin: false },
    },
  },
})
