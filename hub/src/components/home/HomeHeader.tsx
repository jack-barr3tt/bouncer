import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useLogout } from '../../api/generated.ts'
import { loadRegistry } from '../../registry.ts'
import { setSession, useSession } from '../../hooks/useSession.ts'

export default function HomeHeader() {
  const queryClient = useQueryClient()
  const logout = useLogout()
  const { session } = useSession()
  const registry = useQuery({ queryKey: ['apps.yaml'], queryFn: loadRegistry })
  const visible = session ? (registry.data ?? []).filter((app) => session.apps.includes(app.slug)) : []

  async function signOut() {
    await logout.mutateAsync()
    setSession(queryClient, null)
    window.location.assign('/login')
  }

  return (
    <header className="flex max-w-xl flex-col gap-3">
      <p className="text-sm font-medium text-stone-500">Personal</p>
      <div className="flex items-end justify-between gap-4">
        <h1 className="text-4xl font-semibold tracking-tight">Apps</h1>
        {session ? (
          <div className="flex items-center gap-3 text-sm">
            {session.role === 'admin' ? (
              <a className="font-medium underline" href="/admin">
                Admin
              </a>
            ) : null}
            <button className="font-medium underline" type="button" onClick={() => void signOut()}>
              Sign out
            </button>
          </div>
        ) : (
          <a className="text-sm font-medium underline" href="/login">
            Sign in
          </a>
        )}
      </div>
      <p className="text-base text-stone-600">
        {session?.kind === 'temporary'
          ? `Signed in as ${session.nickname}.`
          : session
            ? `Signed in as ${session.username}.`
            : 'Sign in to open your apps.'}
      </p>
      {session && visible.length > 0 ? (
        <p className="text-sm text-stone-500">{visible.length === 1 ? '1 app' : `${visible.length} apps`}</p>
      ) : null}
    </header>
  )
}
