import { Link, createFileRoute } from '@tanstack/react-router'
import AccessCodesSection from '../components/admin/AccessCodesSection.tsx'
import AccountsSection from '../components/admin/AccountsSection.tsx'
import DeploySection from '../components/admin/DeploySection.tsx'
import { useSession } from '../hooks/useSession.ts'

export const Route = createFileRoute('/admin')({
  component: Admin,
})

function Admin() {
  const { session } = useSession()

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
        <DeploySection />
        <AccountsSection />
        <AccessCodesSection />
      </main>
    </div>
  )
}
