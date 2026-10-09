import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { getGetSessionQueryKey, useGetSession, type Session } from '../api/generated.ts'
import { isUnauthorized } from '../api/mutator.ts'
import { watchAccess } from '../registry.ts'

export function useSession() {
  const sessionQuery = useGetSession({ query: { retry: false, retryOnMount: false } })
  const signedOut = isUnauthorized(sessionQuery.error)
  const session = signedOut ? null : (sessionQuery.data ?? null)
  const error =
    sessionQuery.isError && !signedOut && sessionQuery.error instanceof Error ? sessionQuery.error.message : ''
  return { session, error, isPending: sessionQuery.isPending }
}

export function setSession(queryClient: QueryClient, session: Session | null) {
  if (session) queryClient.setQueryData(getGetSessionQueryKey(), session)
  else queryClient.removeQueries({ queryKey: getGetSessionQueryKey() })
}

export function useWatchAccess(session: Session | null) {
  const queryClient = useQueryClient()
  const identity = session ? `${session.kind}:${session.username ?? ''}:${session.nickname ?? ''}` : ''

  useEffect(() => {
    if (!identity) return
    return watchAccess(
      (allowed) => {
        queryClient.setQueryData<Session>(getGetSessionQueryKey(), (current) =>
          current ? { ...current, apps: allowed } : current,
        )
      },
      () => {
        window.location.assign('/login')
      },
    )
  }, [identity, queryClient])
}
