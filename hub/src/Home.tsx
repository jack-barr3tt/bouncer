import { useEffect, useState } from 'react'
import { useLogout, type Session } from './api/generated.ts'
import type { AppEntry } from './registry.ts'

type Availability = 'up' | 'down'

function requestSignal(parent: AbortSignal): AbortSignal {
  if (typeof AbortSignal.any === 'function' && typeof AbortSignal.timeout === 'function') {
    return AbortSignal.any([parent, AbortSignal.timeout(5000)])
  }
  return parent
}

async function checkApp(path: string, parent: AbortSignal): Promise<boolean | null> {
  let url: URL
  try {
    url = new URL(path, window.location.href)
  } catch {
    return false
  }
  const sameOrigin = url.origin === window.location.origin
  try {
    const response = await fetch(url, { signal: requestSignal(parent), cache: 'no-store' })
    if (!sameOrigin && response.type === 'opaque') return null
    return response.ok
  } catch {
    if (parent.aborted) return null
    return sameOrigin ? false : null
  }
}

export default function Home({
  session,
  apps,
  appsReady,
  onSession,
}: {
  session: Session | null
  apps: AppEntry[]
  appsReady: boolean
  onSession: (session: Session | null) => void
}) {
  const [availability, setAvailability] = useState<Record<string, Availability>>({})
  const logout = useLogout()
  const visible = session ? apps.filter((app) => session.apps.includes(app.slug)) : []
  const sole = session?.kind === 'temporary' && appsReady && visible.length === 1 ? visible[0].path : ''

  useEffect(() => {
    if (!sole) return
    window.location.replace(sole)
  }, [sole])

  useEffect(() => {
    if (!session || !import.meta.env.PROD || sole) return
    const controller = new AbortController()
    const granted = apps.filter((app) => session.apps.includes(app.slug))
    for (const app of granted) {
      void checkApp(app.path, controller.signal).then((up) => {
        if (controller.signal.aborted || up === null) return
        setAvailability((current) => ({ ...current, [app.slug]: up ? 'up' : 'down' }))
      })
    }
    return () => controller.abort()
  }, [session, apps, sole])

  async function signOut() {
    await logout.mutateAsync()
    onSession(null)
    window.location.assign('/login')
  }

  if (session?.kind === 'temporary' && (!appsReady || sole)) {
    return <p className="px-6 py-16 text-stone-600">Loading…</p>
  }

  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <main className="mx-auto max-w-5xl px-4 py-12 sm:px-6 sm:py-16">
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
            <p className="text-sm text-stone-500">
              {visible.length === 1 ? '1 app' : `${visible.length} apps`}
            </p>
          ) : null}
        </header>

        <section className="mt-8">
          {!session ? (
            <div className="max-w-xl rounded-2xl border border-stone-200 bg-white px-5 py-8">
              <p className="font-medium">Sign in to see your apps</p>
              <a className="mt-4 inline-block text-sm font-medium underline" href="/login">
                Sign in
              </a>
            </div>
          ) : null}

          {session && visible.length === 0 ? (
            <div className="max-w-xl rounded-2xl border border-dashed border-stone-300 bg-white px-5 py-8">
              <p className="font-medium">No apps yet</p>
              <p className="mt-2 text-sm text-stone-600">Nothing has been turned on for this account.</p>
            </div>
          ) : null}

          {session && visible.length > 0 ? (
            <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {visible.map((app) => {
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
      </main>
    </div>
  )
}
