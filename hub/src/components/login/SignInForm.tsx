import { Alert, Button, Card, Label, TextInput } from 'flowbite-react'
import { useState, type SubmitEvent } from 'react'
import { useLogin, type Session } from '../../api/generated.ts'
import { visitorId } from '../../fingerprint.ts'

function safeNext(value: string | null, session: Session): string {
  if (!value) return '/'
  if (value.startsWith('/') && !value.startsWith('//')) {
    if (session.role === 'admin') return value
    const [prefix, slug] = value.split('/').filter(Boolean)
    if (prefix !== 'apps' || !slug || !session.apps.includes(slug)) return '/'
    return value
  }
  let url: URL
  let hub: URL
  try {
    url = new URL(value)
    hub = new URL(session.hub)
  } catch {
    return '/'
  }
  if (url.protocol !== hub.protocol || url.port !== hub.port) return '/'
  const suffix = `.${hub.hostname}`
  if (!url.hostname.endsWith(suffix)) return '/'
  const slug = url.hostname.slice(0, -suffix.length)
  if (!slug || slug.includes('.')) return '/'
  if (session.role !== 'admin' && !session.apps.includes(slug)) return '/'
  return url.toString()
}

export default function SignInForm() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
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
      const next = new URLSearchParams(window.location.search).get('next')
      window.location.assign(safeNext(next, session))
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Could not sign in.')
      setPending(false)
    }
  }

  return (
    <Card className="h-full">
      <form className="flex h-full flex-col gap-4" onSubmit={(event) => void onSubmit(event)}>
        <div>
          <Label htmlFor="username">Username</Label>
          <TextInput
            id="username"
            className="mt-1"
            autoComplete="username"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            required
          />
        </div>
        <div>
          <Label htmlFor="password">Password</Label>
          <TextInput
            id="password"
            className="mt-1"
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </div>
        {error ? <Alert color="failure">{error}</Alert> : null}
        <Button className="mt-auto" type="submit" disabled={pending}>
          {pending ? 'Signing in…' : 'Sign in'}
        </Button>
      </form>
    </Card>
  )
}
