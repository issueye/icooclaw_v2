import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import path from "path";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "chalk": path.resolve(__dirname, "./src/mocks/chalk.js"),
    },
  },
  define: {
    'process.env': {},
    'global': 'globalThis',
  },
  build: {
    rollupOptions: {
      external: [],
      output: {
        globals: {},
      },
    },
    commonjsOptions: {
      transformMixedEsModules: true,
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:16789",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:16789",
        ws: true,
      },
    }
  }
})
