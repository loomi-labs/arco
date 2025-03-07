import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

// https://vitejs.dev/config/
export default defineConfig({
  build: {
    target: 'esnext'
  },
  plugins: [
    vue(),
    tailwindcss(),
  ]
});
