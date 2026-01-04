import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import fs from 'fs'

// Read package.json to get version
const packageJson = JSON.parse(fs.readFileSync('./package.json', 'utf-8'))

export default defineConfig({
  define: {
    '__APP_VERSION__': JSON.stringify(packageJson.version),
  },
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
      '/system': 'http://localhost:8000',
      '/settings': 'http://localhost:8000',
      '/auth': 'http://localhost:8000',
    },
  },
})
