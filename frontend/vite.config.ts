import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:9093',
        changeOrigin: true,
      },
      '/login': {
        target: 'http://localhost:9093',
        changeOrigin: true,
      },
      '/logout': {
        target: 'http://localhost:9093',
        changeOrigin: true,
      },
      '/init': {
        target: 'http://localhost:9093',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
