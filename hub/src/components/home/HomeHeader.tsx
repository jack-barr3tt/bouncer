import { useApps, useSession } from '../../hooks/useSession.ts'

export default function HomeHeader() {
  const { session, isPending } = useSession()
  const { apps } = useApps()

  return (
    <header className="flex max-w-xl flex-col gap-3">
      <p className="text-sm font-medium text-stone-500">Personal</p>
      <h1 className="text-4xl font-semibold tracking-tight">Apps</h1>
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
