import { useState, type SubmitEvent } from 'react'
import { useLogin, type Session } from './api/generated.ts'
import { visitorId } from './fingerprint.ts'

function safeNext(value: string | null, session: Session): string {
  if (!value || !value.startsWith('/') || value.startsWith('//') || value.includes('://')) return '/'
  if (session.role === 'admin') return value
  const [prefix, slug] = value.split('/').filter(Boolean)
  if (prefix !== 'apps' || !slug || !session.apps.includes(slug)) return '/'
  return value
}

export default function Login({ onSession }: { onSession: (session: Session) => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  const login = useLogin()

  async function onSubmit(event: SubmitEvent) {
    event.preventDefault()
    setError('')
    setPending(true)
    try {
      const fingerprint = await visitorId().catch(() => '')
      const session = await login.mutateAsync({
        data: { username, password, fingerprint: fingerprint || undefined },
      })
      onSession(session)
      const next = new URLSearchParams(window.location.search).get('next')
      window.location.assign(safeNext(next, session))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Could not sign in.')
      setPending(false)
    }
  }

  function openCode(event: SubmitEvent) {
    event.preventDefault()
    const normalized = code.toUpperCase().replace(/[\s-]/g, '')
    if (!normalized) return
    window.location.assign(`/code/${normalized}`)
  }

  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <main className="mx-auto flex max-w-md flex-col gap-8 px-4 py-16">
        <header>
          <p className="text-sm font-medium text-stone-500">Personal</p>
          <h1 className="mt-2 text-4xl font-semibold tracking-tight">Sign in</h1>
        </header>
        <form className="rounded-2xl border border-stone-200 bg-white p-5" onSubmit={(event) => void onSubmit(event)}>
          <label className="block text-sm font-medium" htmlFor="username">
            Username
          </label>
          <input
            id="username"
            className="mt-1 w-full rounded-xl border border-stone-300 px-3 py-2"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            required
          />
          <label className="mt-4 block text-sm font-medium" htmlFor="password">
            Password
          </label>
          <input
            id="password"
            className="mt-1 w-full rounded-xl border border-stone-300 px-3 py-2"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
          {error ? (
            <p className="mt-4 text-sm text-red-700" role="alert">
              {error}
            </p>
          ) : null}
          <button
            className="mt-5 rounded-xl bg-stone-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
            type="submit"
            disabled={pending}
          >
            {pending ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
        <form className="rounded-2xl border border-stone-200 bg-white p-5" onSubmit={openCode}>
          <label className="block text-sm font-medium" htmlFor="code">
            Have a code?
          </label>
          <p className="mt-1 text-sm text-stone-600">This opens the join page. It does not sign you in.</p>
          <input
            id="code"
            className="mt-3 w-full rounded-xl border border-stone-300 px-3 py-2 font-mono uppercase"
            value={code}
            onChange={(event) => setCode(event.target.value)}
            autoCapitalize="characters"
          />
          <button className="mt-4 rounded-xl border border-stone-300 px-4 py-2 text-sm font-medium" type="submit">
            Continue
          </button>
        </form>
      </main>
    </div>
  )
}
