import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,  // 支持 WebSocket 代理
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})

