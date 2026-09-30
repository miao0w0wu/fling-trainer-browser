import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: (id: string) => {
          if (id.includes('node_modules/antd') || id.includes('@ant-design/icons')) {
            return 'antd'
          }
          if (id.includes('node_modules/react')) {
            return 'react'
          }
          if (id.includes('node_modules/@wailsio')) {
            return 'wails'
          }
          return undefined
        },
      },
    },
  },
  plugins: [react(), wails("./bindings")],
});
