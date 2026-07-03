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
      '/v1/deployments': {
        target: 'http://localhost:8085',
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
