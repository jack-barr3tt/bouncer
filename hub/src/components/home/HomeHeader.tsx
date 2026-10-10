import { Badge } from 'flowbite-react'
import { useApps, useSession } from '../../hooks/useSession.ts'

export default function HomeHeader() {
  const { session, isPending } = useSession()
  const { apps } = useApps()

  return (
    <header className="flex max-w-xl flex-col gap-3">
      <p className="text-xs font-semibold tracking-[0.22em] text-pink-300 uppercase">Personal</p>
      <h1 className="text-5xl font-extrabold">Apps</h1>
      {!isPending ? (
        <p className="text-base text-pink-100/75">
          {session?.kind === 'temporary'
            ? `Signed in as ${session.nickname}.`
            : session
              ? `Signed in as ${session.username}.`
              : 'Sign in to open your apps.'}
        </p>
      ) : null}
      {session && apps.length > 0 ? (
        <Badge className="w-fit" color="pink">
          {apps.length === 1 ? '1 app' : `${apps.length} apps`}
        </Badge>
      ) : null}
    </header>
  )
}
