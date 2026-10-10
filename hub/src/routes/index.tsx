import { createFileRoute } from '@tanstack/react-router'
import { Alert } from 'flowbite-react'
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
      {error ? <Alert color="failure">{error}</Alert> : null}
      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 sm:py-12">
        <HomeHeader />
        {sessionPending || (session && appsPending) ? null : <AppList />}
      </main>
    </>
  )
}
