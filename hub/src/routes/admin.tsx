import { Link, createFileRoute } from '@tanstack/react-router'
import AccessCodesSection from '../components/admin/AccessCodesSection.tsx'
import AccountsSection from '../components/admin/AccountsSection.tsx'
import DeploySection from '../components/admin/DeploySection.tsx'
import { useApps, useSession } from '../hooks/useSession.ts'

export const Route = createFileRoute('/admin')({
  component: Admin,
})

function Admin() {
  const { session, isPending, error } = useSession()
  const { apps, isPending: appsPending, error: appsError } = useApps()

  if (isPending || (session?.role === 'admin' && appsPending)) {
    return <main className="mx-auto max-w-3xl px-4 py-16" />
  }

  if (!session || session.role !== 'admin') {
    return (
      <main className="mx-auto max-w-md px-4 py-16">
        <p>Admin is only available to an admin account.</p>
        <Link className="mt-4 inline-block text-sm font-medium underline" to={session ? '/' : '/login'}>
          {session ? 'All apps' : 'Sign in'}
        </Link>
      </main>
    )
  }

  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <main className="mx-auto max-w-3xl px-4 py-12">
        <Link className="text-sm font-medium underline" to="/">
          All apps
        </Link>
        <h1 className="mt-3 text-4xl font-semibold tracking-tight">Admin</h1>
        {error || appsError ? (
          <p className="mt-4 text-sm text-red-700" role="alert">
            {error || appsError}
          </p>
        ) : null}
        <DeploySection />
        <AccountsSection apps={apps} />
        <AccessCodesSection apps={apps} />
      </main>
    </div>
  )
}
