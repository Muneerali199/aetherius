import path from "path"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"
import { inspectAttr } from 'kimi-plugin-inspect-react'

export default defineConfig({
  base: './',
  plugins: [inspectAttr(), react()],
  server: {
    port: parseInt(process.env.VITE_PORT || '3000'),
    proxy: {
      '/v1/auth': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/v1/nodes': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/v1/ssh-keys': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/v1/workspace': {
        target: 'http://localhost:8082',
        changeOrigin: true,
        ws: true,
      },
      '/v1/deployments': {
        target: 'http://localhost:8083',
        changeOrigin: true,
      },
      '/v1/listings': {
        target: 'http://localhost:8086',
        changeOrigin: true,
      },
      '/v1/orders': {
        target: 'http://localhost:8086',
        changeOrigin: true,
      },
      '/v1/billing': {
        target: 'http://localhost:8087',
        changeOrigin: true,
      },
      '/v1/user': {
        target: 'http://localhost:8084',
        changeOrigin: true,
      },
      '/v1/storage': {
        target: 'http://localhost:8088',
        changeOrigin: true,
      },
      '/v1/notifications': {
        target: 'http://localhost:8089',
        changeOrigin: true,
      },
      '/v1/networking': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
      '/v1/ai': {
        target: 'http://localhost:8091',
        changeOrigin: true,
      },
      '/v1/monitoring': {
        target: 'http://localhost:8092',
        changeOrigin: true,
      },
      '/v1/support': {
        target: 'http://localhost:8093',
        changeOrigin: true,
      },
      '/v1/payments': {
        target: 'http://localhost:8094',
        changeOrigin: true,
      },
      '/ws': {
        target: process.env.VITE_WS_URL || 'ws://localhost:8080',
        ws: true,
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
