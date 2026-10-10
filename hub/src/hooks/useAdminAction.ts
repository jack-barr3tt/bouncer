import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import { reportError } from '../toasts.tsx'

const adminPaths = ['/api/users', '/api/access-codes', '/api/deploy', '/api/apps']

function isAdminQuery(key: unknown) {
  return typeof key === 'string' && adminPaths.some((path) => key === path || key.startsWith(`${path}/`))
}

export function invalidateAdmin(queryClient: QueryClient) {
  return queryClient.invalidateQueries({
    predicate: (query) => isAdminQuery(query.queryKey[0]),
  })
}

export function useAdminAction() {
  const queryClient = useQueryClient()

  async function run(action: () => Promise<unknown>) {
    try {
      await action()
      await invalidateAdmin(queryClient)
    } catch (caught) {
      reportError(caught)
    }
  }

  return { run }
}

export type AdminAction = ReturnType<typeof useAdminAction>['run']
