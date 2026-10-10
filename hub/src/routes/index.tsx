import { createFileRoute } from '@tanstack/react-router'
import { useEffect } from 'react'
import AccessCodesSection from '../components/admin/AccessCodesSection.tsx'
import AppList from '../components/home/AppList.tsx'
import HomeHeader from '../components/home/HomeHeader.tsx'
import { useApps, useSession } from '../hooks/useSession.ts'
import { useToastError } from '../toasts.tsx'

export const Route = createFileRoute('/')({
  component: Home,
})

function Home() {
  const { error: sessionError, session, isPending: sessionPending } = useSession()
  const { apps, error: appsError, isPending: appsPending } = useApps()
  const error = sessionError || appsError
  useToastError(error)
  const shareable = session?.role === 'user' ? apps.filter((app) => session.shareApps?.includes(app.slug)) : []

  useEffect(() => {
    if (session?.kind !== 'temporary' || appsPending || appsError || apps.length !== 1) return
    window.location.assign(apps[0].path)
  }, [apps, appsError, appsPending, session])

  return (
    <>
      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 sm:py-12">
        <HomeHeader />
        {sessionPending || (session && appsPending) ? null : <AppList />}
        {shareable.length > 0 ? <AccessCodesSection apps={shareable} heading="Share access" /> : null}
      </main>
    </>
  )
}
