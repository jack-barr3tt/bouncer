import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import AppList from '../components/home/AppList.tsx'
import HomeHeader from '../components/home/HomeHeader.tsx'
import { useApps, useSession } from '../hooks/useSession.ts'

export const Route = createFileRoute('/')({
  component: Home,
})

function Home() {
  const { error: sessionError, session, isPending: sessionPending } = useSession()
  const { apps, error: appsError, isPending: appsPending } = useApps()
  const error = sessionError || appsError

  useEffect(() => {
    if (session?.kind !== 'temporary' || appsPending || appsError || apps.length !== 1) return
    window.location.assign(apps[0].path)
  }, [apps, appsError, appsPending, session])

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
          {sessionPending || (session && appsPending) ? null : <AppList />}
        </main>
      </div>
    </>
  )
}
