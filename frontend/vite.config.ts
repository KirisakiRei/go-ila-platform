import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { fileURLToPath, URL } from 'node:url'

/**
 * Alias @/ menunjuk ke src/ agar import tidak berupa rentetan ../../..
 * Proxy dev server meneruskan /api dan /health ke backend-api sehingga
 * browser tidak terkena CORS saat pengembangan lokal.
 */
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    chunkSizeWarningLimit: 600,
    rollupOptions: {
      output: {
        // Vite 8 memakai rolldown; manualChunks kini hanya menerima fungsi.
        manualChunks(id: string) {
          if (id.includes('node_modules/@tanstack')) return 'query'
          if (id.includes('node_modules/react')) return 'vendor'
          return undefined
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./vitest.setup.ts'],
    css: false,
  },
})
