import { Navigate, createFileRoute } from '@tanstack/react-router'
import AccessCodesSection from '../components/admin/AccessCodesSection.tsx'
import AccountsSection from '../components/admin/AccountsSection.tsx'
import AppsSection from '../components/admin/AppsSection.tsx'
import DeploySection from '../components/admin/DeploySection.tsx'
import { useApps, useSession } from '../hooks/useSession.ts'
import { useToastError } from '../toasts.tsx'

export const Route = createFileRoute('/admin')({
  component: Admin,
})

function Admin() {
  const { session, isPending, error } = useSession()
  const { apps, isPending: appsPending, error: appsError } = useApps()
  useToastError(error || appsError)

  if (isPending || (session?.role === 'admin' && appsPending)) {
    return <main className="mx-auto max-w-3xl px-4 py-8" />
  }

  if (session?.role !== 'admin') {
    return <Navigate to="/" replace />
  }

  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <p className="accent text-xs font-semibold tracking-[0.22em] uppercase">Site</p>
      <h1 className="mt-2 text-5xl font-extrabold">Admin</h1>
      <DeploySection />
      <AppsSection apps={apps} />
      <AccountsSection apps={apps} />
      <AccessCodesSection apps={apps} />
    </main>
  )
}
