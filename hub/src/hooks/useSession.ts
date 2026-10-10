import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { getGetSessionQueryKey, getListAppsQueryKey, getSession, listApps, useGetSession, useListApps, type Session } from '../api/generated.ts'
import { isUnauthorized } from '../api/mutator.ts'

export function useSession() {
  const query = useGetSession({
    query: {
      retry: false,
      enabled: typeof window !== 'undefined',
      queryFn: () => getSession(),
    },
  })
  const signedOut = isUnauthorized(query.error)
  const error = query.error && !signedOut ? query.error.message : ''
  return { session: signedOut ? null : (query.data ?? null), error, isPending: query.isPending }
}

export function useApps() {
  const query = useListApps({
    query: {
      enabled: typeof window !== 'undefined',
      queryFn: () => listApps(),
    },
  })
  const error = query.error instanceof Error ? query.error.message : ''
  return { apps: query.data?.apps ?? [], error, isPending: query.isPending }
}

function watchAccess(onAccess: (apps: string[]) => void, onEnded: () => void): () => void {
  const stream = new EventSource('/api/access/stream')
  stream.addEventListener('access', (event) => {
    const data = JSON.parse((event as MessageEvent).data) as { apps?: string[] }
    onAccess(data.apps ?? [])
  })
  stream.addEventListener('session_ended', () => {
    onEnded()
  })
  return () => stream.close()
}

export function useWatchAccess(session: Session | null) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const identity = session ? `${session.kind}:${session.username ?? ''}:${session.nickname ?? ''}` : ''

  useEffect(() => {
    if (!identity) return
    return watchAccess(
      () => {
        void queryClient.invalidateQueries({ queryKey: getGetSessionQueryKey() })
        void queryClient.invalidateQueries({ queryKey: getListAppsQueryKey() })
      },
      () => {
        queryClient.removeQueries({ queryKey: getGetSessionQueryKey() })
        queryClient.removeQueries({ queryKey: getListAppsQueryKey() })
        void navigate({ to: '/login' })
      },
    )
  }, [identity, navigate, queryClient])
}
