import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import Admin from './Admin.tsx'
import { getGetSessionQueryKey, useGetSession, type Session } from './api/generated.ts'
import { isUnauthorized } from './api/mutator.ts'
import Home from './Home.tsx'
import Join from './Join.tsx'
import Login from './Login.tsx'
import { loadRegistry, watchAccess } from './registry.ts'

function screen(path: string): 'login' | 'join' | 'admin' | 'home' {
  if (path === '/login') return 'login'
  if (path === '/admin' || path.startsWith('/admin/')) return 'admin'
  if (path === '/code' || path.startsWith('/code/')) return 'join'
  return 'home'
}

export default function App() {
  const queryClient = useQueryClient()
  const [path, setPath] = useState(window.location.pathname)
  const sessionQuery = useGetSession({ query: { retry: false } })
  const registryQuery = useQuery({ queryKey: ['apps.yaml'], queryFn: loadRegistry })
  const signedOut = isUnauthorized(sessionQuery.error)
  const session = signedOut ? null : sessionQuery.data
  const sessionError =
    sessionQuery.isError && !signedOut && sessionQuery.error instanceof Error ? sessionQuery.error.message : ''
  const registryError = registryQuery.error instanceof Error ? registryQuery.error.message : ''
  const error = sessionError || registryError

  useEffect(() => {
    const onPop = () => setPath(window.location.pathname)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

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

  function onSession(next: Session | null) {
    if (next) queryClient.setQueryData(getGetSessionQueryKey(), next)
    else queryClient.removeQueries({ queryKey: getGetSessionQueryKey() })
  }

  if (sessionQuery.isPending && !error) {
    return <p className="px-6 py-16 text-stone-600">Loading…</p>
  }

  const view = screen(path)
  if (view === 'login') return <Login onSession={onSession} />
  if (view === 'join') return <Join />
  if (view === 'admin') {
    if (!session || session.role !== 'admin') {
      return (
        <main className="mx-auto max-w-md px-4 py-16">
          <p>Admin is only available to an admin account.</p>
          <a className="mt-4 inline-block text-sm font-medium underline" href={session ? '/' : '/login'}>
            {session ? 'All apps' : 'Sign in'}
          </a>
        </main>
      )
    }
    return <Admin apps={registryQuery.data ?? []} />
  }

  return (
    <>
      {error ? (
        <p className="bg-red-50 px-6 py-3 text-sm text-red-800" role="alert">
          {error}
        </p>
      ) : null}
      <Home
        session={session ?? null}
        apps={registryQuery.data ?? []}
        appsReady={registryQuery.isFetched}
        onSession={onSession}
      />
    </>
  )
}
