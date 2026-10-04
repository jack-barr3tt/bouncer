import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'

const hubDir = path.dirname(fileURLToPath(import.meta.url))

function registryPath(): string {
  if (process.env.APPS_REGISTRY) return path.resolve(process.env.APPS_REGISTRY)
  const site = path.resolve(hubDir, '../apps.yaml')
  try {
    readFileSync(site)
    return site
  } catch {
    return path.resolve(hubDir, '../scaffold/apps.yaml')
  }
}

function readRegistry(): string {
  const file = registryPath()
  try {
    return readFileSync(file, 'utf8')
  } catch (error) {
    const detail = error instanceof Error ? error.message : 'unknown error'
    throw new Error(`Could not read the app registry at ${file}: ${detail}`)
  }
}

function registryPlugin(): Plugin {
  return {
    name: 'app-registry',
    configureServer(server) {
      server.middlewares.use('/apps.yaml', (request, response, next) => {
        if (request.method !== 'GET' && request.method !== 'HEAD') {
          next()
          return
        }
        response.setHeader('Content-Type', 'application/yaml')
        response.setHeader('Cache-Control', 'no-store')
        response.end(readRegistry())
      })
    },
    generateBundle() {
      this.emitFile({
        type: 'asset',
        fileName: 'apps.yaml',
        source: readRegistry(),
      })
    },
  }
}

export default defineConfig({
  base: '/',
  plugins: [react(), tailwindcss(), registryPlugin()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/apps': 'http://127.0.0.1:8080',
    },
  },
})
