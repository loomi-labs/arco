import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    // Bind IPv4 explicitly: the Wails3 dev asset proxy dials tcp4 only, but with
    // the default host ("localhost") Node may bind ::1 only, causing a white screen
    host: "127.0.0.1"
  },
  build: {
    target: 'esnext'
  },
  plugins: [
    vue(),
    tailwindcss(),
  ]
});
