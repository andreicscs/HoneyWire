import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
  ],
  server: {
    // This tells Vite to forward API requests to your Go backend
    proxy: {
      '/api': {
        target: 'http://localhost:8080', // <-- CHANGE TO YOUR GO BACKEND PORT
        changeOrigin: true,
        secure: false,
      },
      '/login': {
        target: 'http://localhost:8080', // <-- CHANGE TO YOUR GO BACKEND PORT
        changeOrigin: true,
        secure: false,
      },
      '/logout': {
        target: 'http://localhost:8080', // <-- CHANGE TO YOUR GO BACKEND PORT
        changeOrigin: true,
        secure: false,
      }
    }
  }
})