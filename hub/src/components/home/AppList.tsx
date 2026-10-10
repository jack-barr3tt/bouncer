import { Link } from '@tanstack/react-router'
import { useAppAvailability } from '../../hooks/useAppAvailability.ts'
import { useApps, useSession } from '../../hooks/useSession.ts'

export default function AppList() {
  const { session } = useSession()
  const { apps } = useApps()
  const availability = useAppAvailability(session, apps)

  return (
    <section className="mt-8">
      {!session ? (
        <div className="max-w-xl rounded-2xl border border-stone-200 bg-white px-5 py-8">
          <p className="font-medium">Sign in to see your apps</p>
          <Link className="mt-4 inline-block text-sm font-medium underline" to="/login">
            Sign in
          </Link>
        </div>
      ) : null}

      {session && apps.length === 0 ? (
        <div className="max-w-xl rounded-2xl border border-dashed border-stone-300 bg-white px-5 py-8">
          <p className="font-medium">No apps yet</p>
          <p className="mt-2 text-sm text-stone-600">Nothing has been turned on for this account.</p>
        </div>
      ) : null}

      {session && apps.length > 0 ? (
        <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {apps.map((app) => {
            const down = availability[app.slug] === 'down'
            return (
              <li key={app.slug}>
                <a
                  className={`flex h-full flex-col rounded-2xl border bg-white p-5 hover:border-stone-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-stone-800 ${down ? 'border-amber-300' : 'border-stone-200'}`}
                  href={app.path}
                >
                  <span className="flex size-12 items-center justify-center rounded-xl bg-stone-100 text-sm font-semibold">
                    {app.icon}
                  </span>
                  <span className="mt-4 text-lg font-semibold tracking-tight break-words">{app.name}</span>
                  <span className="mt-1 text-sm leading-6 text-stone-600">{app.description}</span>
                  {down ? <span className="mt-3 text-sm font-medium text-amber-800">Unavailable</span> : null}
                </a>
              </li>
            )
          })}
        </ul>
      ) : null}
    </section>
  )
}
