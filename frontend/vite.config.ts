import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    vueDevTools(),
  ],
  server: {
  proxy: {
    '/api': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
      // /api/account/login → http://127.0.0.1:8080/account/login
      rewrite: (path) => path.replace(/^\/api/, ''),
    },
    // 视频和头像走 /static，前缀要保留，这轮还用不到
    '/static': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
  },
},
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
