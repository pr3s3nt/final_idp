/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// ADR-017: the app lives under /ui/ and calls the Go backend on the same
// origin. In development Vite proxies /api to `idp serve`.
const apiOrigin = process.env.IDP_API_ORIGIN ?? 'http://127.0.0.1:8088';

export default defineConfig({
  base: '/ui/',
  plugins: [react()],
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: { '/api': { target: apiOrigin, changeOrigin: false } },
  },
  build: { outDir: 'dist', emptyOutDir: true },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    restoreMocks: true,
  },
});
