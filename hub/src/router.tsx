import { createRouter } from '@tanstack/react-router'
import { routeTree } from './routeTree.gen.ts'

export function getRouter() {
  const router = createRouter({
    routeTree,
    scrollRestoration: true,
    defaultStaleTime: 0,
  })

  return router
}

declare module '@tanstack/react-router' {
  interface Register {
    router: ReturnType<typeof getRouter>
  }
}
