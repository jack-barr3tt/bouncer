import AccessCodesSection from '../components/admin/AccessCodesSection.tsx'
import AccountsSection from '../components/admin/AccountsSection.tsx'
import DeploySection from '../components/admin/DeploySection.tsx'
import { useSession } from '../hooks/useSession.ts'

export default function Admin() {
  const { session } = useSession()

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

  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <main className="mx-auto max-w-3xl px-4 py-12">
        <a className="text-sm font-medium underline" href="/">
          All apps
        </a>
        <h1 className="mt-3 text-4xl font-semibold tracking-tight">Admin</h1>
        <DeploySection />
        <AccountsSection />
        <AccessCodesSection />
      </main>
    </div>
  )
}
