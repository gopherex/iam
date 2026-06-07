import path from 'node:path';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // Dev: proxy API to the Go server so the SPA can call /v1/* and /mgmt/*.
      '/v1': 'http://localhost:8080',
      '/mgmt': 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
  },
});
