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
    // Dua entri terpisah: situs publik dan panel pelanggan. Bundel dashboard
    // tidak ikut terunduh pengunjung tanpa akun (plans/11 J5b).
    rollupOptions: {
      input: {
        main: fileURLToPath(new URL('./index.html', import.meta.url)),
        app: fileURLToPath(new URL('./app.html', import.meta.url)),
      },
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
