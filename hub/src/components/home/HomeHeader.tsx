import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { getGetSessionQueryKey, getListAppsQueryKey, useLogout } from '../../api/generated.ts'
import { useApps, useSession } from '../../hooks/useSession.ts'

export default function HomeHeader() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const logout = useLogout()
  const { session, isPending } = useSession()
  const { apps } = useApps()

  async function signOut() {
    await logout.mutateAsync()
    queryClient.removeQueries({ queryKey: getGetSessionQueryKey() })
    queryClient.removeQueries({ queryKey: getListAppsQueryKey() })
    await navigate({ to: '/login' })
  }

  return (
    <header className="flex max-w-xl flex-col gap-3">
      <p className="text-sm font-medium text-stone-500">Personal</p>
      <div className="flex items-end justify-between gap-4">
        <h1 className="text-4xl font-semibold tracking-tight">Apps</h1>
        {!isPending && session ? (
          <div className="flex items-center gap-3 text-sm">
            {session.role === 'admin' ? (
              <Link className="font-medium underline" to="/admin">
                Admin
              </Link>
            ) : null}
            <button className="font-medium underline" type="button" onClick={() => void signOut()}>
              Sign out
            </button>
          </div>
        ) : null}
        {!isPending && !session ? (
          <Link className="text-sm font-medium underline" to="/login">
            Sign in
          </Link>
        ) : null}
      </div>
      {!isPending ? (
        <p className="text-base text-stone-600">
          {session?.kind === 'temporary'
            ? `Signed in as ${session.nickname}.`
            : session
              ? `Signed in as ${session.username}.`
              : 'Sign in to open your apps.'}
        </p>
      ) : null}
      {session && apps.length > 0 ? (
        <p className="text-sm text-stone-500">{apps.length === 1 ? '1 app' : `${apps.length} apps`}</p>
      ) : null}
    </header>
  )
}
