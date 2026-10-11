import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect } from 'react'
import { getGetSessionQueryKey, getListAppsQueryKey, getSession, listApps, useGetSession, useListApps, type Session } from '../api/generated.ts'
import { isUnauthorized } from '../api/mutator.ts'

export function useSession() {
  const query = useGetSession({
    query: {
      retry: false,
      retryOnMount: false,
      staleTime: 10_000,
      enabled: typeof window !== 'undefined',
      queryFn: () => getSession(),
    },
  })
  const signedOut = isUnauthorized(query.error)
  const error = query.error && !signedOut ? query.error.message : ''
  return { session: signedOut ? null : (query.data ?? null), error, isPending: !query.isFetched }
}

export function useApps() {
  const query = useListApps({
    query: {
      retryOnMount: false,
      staleTime: 10_000,
      enabled: typeof window !== 'undefined',
      queryFn: () => listApps(),
    },
  })
  const error = query.error instanceof Error ? query.error.message : ''
  return { apps: query.data?.apps ?? [], error, isPending: !query.isFetched }
}

function watchAccess(onAccess: (apps: string[]) => void, onEnded: () => void): () => void {
  let closed = false
  let stream: EventSource | null = null
  let attempt = 0
  let timer = 0

  const connect = () => {
    if (closed) return
    stream = new EventSource('/api/access/stream')
    stream.addEventListener('open', () => {
      attempt = 0
    })
    stream.addEventListener('access', (event) => {
      const data = JSON.parse((event as MessageEvent).data) as { apps?: string[] }
      onAccess(data.apps ?? [])
    })
    stream.addEventListener('session_ended', () => {
      closed = true
      window.clearTimeout(timer)
      stream?.close()
      onEnded()
    })
    stream.onerror = () => {
      stream?.close()
      if (closed) return
      const delay = Math.min(30_000, 1000 * 2 ** attempt)
      attempt = Math.min(attempt + 1, 5)
      timer = window.setTimeout(connect, delay)
    }
  }

  connect()
  return () => {
    closed = true
    window.clearTimeout(timer)
    stream?.close()
  }
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
