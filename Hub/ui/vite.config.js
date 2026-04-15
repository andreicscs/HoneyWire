import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig(({ mode }) => {
  // Load env variables (falls back to 8080 if HW_PORT isn't set locally)
  const env = loadEnv(mode, process.cwd(), '')
  const backendUrl = `http://localhost:${env.HW_PORT || '8080'}`

  return {
    plugins: [
      vue(),
      tailwindcss(),
    ],
    server: {
      proxy: {
        '/api': {
          target: backendUrl,
          changeOrigin: true,
          secure: false,
          ws: true,
        },
        '/login': {
          target: backendUrl,
          changeOrigin: true,
          secure: false,
        },
        '/logout': {
          target: backendUrl,
          changeOrigin: true,
          secure: false,
        }
      }
    }
  }
})