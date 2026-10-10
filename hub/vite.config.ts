import { tanstackStart } from '@tanstack/react-start/plugin/vite'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'
import flowbiteReact from 'flowbite-react/plugin/vite'

export default defineConfig({
  plugins: [tanstackStart(), react(), tailwindcss(), flowbiteReact()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/apps': 'http://127.0.0.1:8080',
    },
  },
})