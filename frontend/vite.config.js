import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: true,
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/icons': 'http://127.0.0.1:8080'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
