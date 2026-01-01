import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  base: '/ui/',
  build: {
    outDir: '../internal/ui/static/react-app',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/providers': 'http://localhost:8000',
      '/rates': 'http://localhost:8000',
      '/refresh': 'http://localhost:8000',
      '/water': 'http://localhost:8000',
    },
  },
})
