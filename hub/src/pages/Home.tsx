import { useQuery } from '@tanstack/react-query'
import { useEffect } from 'react'
import AppList from '../components/home/AppList.tsx'
import HomeHeader from '../components/home/HomeHeader.tsx'
import { loadRegistry } from '../registry.ts'
import { useSession } from '../hooks/useSession.ts'

export default function Home() {
  const { session, error: sessionError, isPending } = useSession()
  const registry = useQuery({ queryKey: ['apps.yaml'], queryFn: loadRegistry })
  const apps = registry.data ?? []
  const appsReady = registry.isFetched
  const visible = session ? apps.filter((app) => session.apps.includes(app.slug)) : []
  const sole = session?.kind === 'temporary' && appsReady && visible.length === 1 ? visible[0].path : ''
  const registryError = registry.error instanceof Error ? registry.error.message : ''
  const error = sessionError || registryError

  useEffect(() => {
    if (!sole) return
    window.location.replace(sole)
  }, [sole])

  if ((isPending && !error) || (session?.kind === 'temporary' && (!appsReady || sole))) {
    return <p className="px-6 py-16 text-stone-600">Loading…</p>
  }

  return (
    <>
      {error ? (
        <p className="bg-red-50 px-6 py-3 text-sm text-red-800" role="alert">
          {error}
        </p>
      ) : null}
      <div className="min-h-svh bg-stone-100 text-stone-900">
        <main className="mx-auto max-w-5xl px-4 py-12 sm:px-6 sm:py-16">
          <HomeHeader />
          <AppList />
        </main>
      </div>
    </>
  )
}
