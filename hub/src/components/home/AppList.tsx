import { Link } from '@tanstack/react-router'
import { Badge, Card } from 'flowbite-react'
import { useAppAvailability } from '../../hooks/useAppAvailability.ts'
import { useApps, useSession } from '../../hooks/useSession.ts'

export default function AppList() {
  const { session } = useSession()
  const { apps } = useApps()
  const availability = useAppAvailability(session, apps)

  return (
    <section className="mt-8">
      {!session ? (
        <Card className="max-w-xl">
          <p className="font-medium">Sign in to see your apps</p>
          <Link className="text-sm font-medium underline" to="/login">
            Sign in
          </Link>
        </Card>
      ) : null}

      {session && apps.length === 0 ? (
        <Card className="max-w-xl">
          <p className="font-medium">No apps yet</p>
          <p className="text-sm text-stone-600">Nothing has been turned on for this account.</p>
        </Card>
      ) : null}

      {session && apps.length > 0 ? (
        <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {apps.map((app) => {
            const down = availability[app.slug] === 'down'
            return (
              <li key={app.slug}>
                <Card className="h-full" href={app.path}>
                  <span className="flex size-12 items-center justify-center rounded-xl bg-stone-100 text-sm font-semibold">
                    {app.icon}
                  </span>
                  <span className="text-lg font-semibold tracking-tight wrap-break-word">{app.name}</span>
                  <span className="text-sm leading-6 text-stone-600">{app.description}</span>
                  {down ? (
                    <Badge className="w-fit" color="warning">
                      Unavailable
                    </Badge>
                  ) : null}
                </Card>
              </li>
            )
          })}
        </ul>
      ) : null}
    </section>
  )
}
