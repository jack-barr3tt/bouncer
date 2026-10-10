import { Link, createFileRoute } from '@tanstack/react-router'
import { Alert } from 'flowbite-react'
import AccessCodesSection from '../components/admin/AccessCodesSection.tsx'
import AccountsSection from '../components/admin/AccountsSection.tsx'
import DeploySection from '../components/admin/DeploySection.tsx'
import AppShell from '../components/shell/AppShell.tsx'
import { useApps, useSession } from '../hooks/useSession.ts'

export const Route = createFileRoute('/admin')({
  component: Admin,
})

function Admin() {
  const { session, isPending, error } = useSession()
  const { apps, isPending: appsPending, error: appsError } = useApps()

  if (isPending || (session?.role === 'admin' && appsPending)) {
    return (
      <AppShell>
        <main className="mx-auto max-w-3xl px-4 py-8" />
      </AppShell>
    )
  }

  if (!session || session.role !== 'admin') {
    return (
      <AppShell>
        <main className="mx-auto max-w-md px-4 py-8">
          <p>Admin is only available to an admin account.</p>
          <Link className="mt-4 inline-block text-sm font-medium underline" to={session ? '/' : '/login'}>
            {session ? 'All apps' : 'Sign in'}
          </Link>
        </main>
      </AppShell>
    )
  }

  return (
    <AppShell>
      <main className="mx-auto max-w-3xl px-4 py-8">
        <h1 className="text-4xl font-semibold tracking-tight">Admin</h1>
        {error || appsError ? (
          <Alert className="mt-4" color="failure">
            {error || appsError}
          </Alert>
        ) : null}
        <DeploySection />
        <AccountsSection apps={apps} />
        <AccessCodesSection apps={apps} />
      </main>
    </AppShell>
  )
}
