export type Identity = { kind: 'user'; username: string } | { kind: 'temporary'; nickname: string }

type SessionBody = {
  kind?: unknown
  username?: unknown
  nickname?: unknown
}

export async function currentIdentity(): Promise<Identity | null> {
  const response = await fetch('/api/session', { credentials: 'include' })
  if (!response.ok) return null
  const session = (await response.json()) as SessionBody
  if (session.kind === 'temporary' && typeof session.nickname === 'string' && session.nickname !== '') {
    return { kind: 'temporary', nickname: session.nickname }
  }
  if (session.kind === 'user' && typeof session.username === 'string' && session.username !== '') {
    return { kind: 'user', username: session.username }
  }
  return null
}

type AccessBody = {
  apps?: unknown
}

export function watchAccess(slug: string): void {
  const stream = new EventSource('/api/access/stream')
  stream.addEventListener('access', (event) => {
    if (!(event instanceof MessageEvent) || typeof event.data !== 'string') return
    const data = JSON.parse(event.data) as AccessBody
    const apps = Array.isArray(data.apps) ? data.apps : []
    if (!apps.includes(slug)) window.location.assign('/')
  })
  stream.addEventListener('session_ended', () => {
    window.location.assign('/login')
  })
}
