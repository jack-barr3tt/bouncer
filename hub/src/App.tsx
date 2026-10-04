import { useEffect, useState } from 'react'
import Admin from './Admin.tsx'
import { getSession, loadRegistry, watchAccess, type AppEntry, type Session } from './api.ts'
import Home from './Home.tsx'
import Join from './Join.tsx'
import Login from './Login.tsx'

function screen(path: string): 'login' | 'join' | 'admin' | 'home' {
  if (path === '/login') return 'login'
  if (path === '/admin' || path.startsWith('/admin/')) return 'admin'
  if (path === '/code' || path.startsWith('/code/')) return 'join'
  return 'home'
}

export default function App() {
  const [path, setPath] = useState(window.location.pathname)
  const [session, setSession] = useState<Session | null | undefined>(undefined)
  const [apps, setApps] = useState<AppEntry[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    const onPop = () => setPath(window.location.pathname)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    getSession()
      .then((next) => {
        if (!controller.signal.aborted) setSession(next)
      })
      .catch((caught: unknown) => {
        if (controller.signal.aborted) return
        setSession(null)
        setError(caught instanceof Error ? caught.message : 'Could not load the session.')
      })
    loadRegistry()
      .then((next) => {
        if (!controller.signal.aborted) setApps(next)
      })
      .catch((caught: unknown) => {
        if (controller.signal.aborted) return
        setError(caught instanceof Error ? caught.message : 'Could not load the app list.')
      })
    return () => controller.abort()
  }, [])

  const identity = session ? `${session.kind}:${session.username ?? ''}:${session.nickname ?? ''}` : ''
  useEffect(() => {
    if (!identity) return
    return watchAccess(
      (allowed) => setSession((current) => (current ? { ...current, apps: allowed } : current)),
      () => {
        window.location.assign('/login')
      },
    )
  }, [identity])

  if (session === undefined && !error) {
    return <p className="px-6 py-16 text-stone-600">Loading…</p>
  }

  const view = screen(path)
  if (view === 'login') return <Login onSession={setSession} />
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
    return <Admin apps={apps} />
  }

  return (
    <>
      {error ? (
        <p className="bg-red-50 px-6 py-3 text-sm text-red-800" role="alert">
          {error}
        </p>
      ) : null}
      <Home session={session ?? null} apps={apps} onSession={setSession} />
    </>
  )
}
