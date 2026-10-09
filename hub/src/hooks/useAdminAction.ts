import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useState } from 'react'

export function invalidateAdmin(queryClient: QueryClient) {
  return queryClient.invalidateQueries({
    predicate: (query) => {
      const key = query.queryKey[0]
      return (
        key === 'apps.yaml' ||
        (typeof key === 'string' &&
          (key.startsWith('/api/users') || key.startsWith('/api/access-codes') || key === '/api/deploy'))
      )
    },
  })
}

export function useAdminAction() {
  const queryClient = useQueryClient()
  const [error, setError] = useState('')

  async function run(action: () => Promise<unknown>) {
    setError('')
    try {
      await action()
      await invalidateAdmin(queryClient)
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Request failed.')
    }
  }

  return { error, setError, run }
}

export type AdminAction = ReturnType<typeof useAdminAction>['run']
