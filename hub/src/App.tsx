import { useEffect, useState } from 'react'
import { useSession, useWatchAccess } from './hooks/useSession.ts'
import Admin from './pages/Admin.tsx'
import Home from './pages/Home.tsx'
import Join from './pages/Join.tsx'
import Login from './pages/Login.tsx'

function screen(path: string): 'login' | 'join' | 'admin' | 'home' {
  if (path === '/login') return 'login'
  if (path === '/admin' || path.startsWith('/admin/')) return 'admin'
  if (path === '/code' || path.startsWith('/code/')) return 'join'
  return 'home'
}

export default function App() {
  const [path, setPath] = useState(window.location.pathname)
  const { session, error, isPending } = useSession()
  useWatchAccess(session)

  useEffect(() => {
    const onPop = () => setPath(window.location.pathname)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  if (isPending && !error) {
    return <p className="px-6 py-16 text-stone-600">Loading…</p>
  }

  const view = screen(path)
  if (view === 'login') return <Login />
  if (view === 'join') return <Join />
  if (view === 'admin') return <Admin />
  return <Home />
}
