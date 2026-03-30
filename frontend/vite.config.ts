import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import wails from "@wailsio/runtime/plugins/vite";
import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import mpa from 'vite-plugin-mpa'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    wails("./bindings"), 
    tailwindcss(),
    mpa.default({
      open: 'index/index.html',
      scanDir: 'src/pages',
      scanFile: 'main.ts',
      filename: 'index.html'
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, './src'),
    },
  },
});
