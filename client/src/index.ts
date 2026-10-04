import { accessDestination } from './leave.js'

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
  hub?: unknown
}

function readEvent(event: Event): AccessBody | null {
  if (!(event instanceof MessageEvent) || typeof event.data !== 'string') return null
  return JSON.parse(event.data) as AccessBody
}

export function watchAccess(slug: string): void {
  const stream = new EventSource('/api/access/stream')
  stream.addEventListener('access', (event) => {
    const data = readEvent(event)
    if (!data) return
    const dest = accessDestination('access', data, slug)
    if (dest) window.location.assign(dest)
  })
  stream.addEventListener('session_ended', (event) => {
    const data = readEvent(event) ?? {}
    window.location.assign(accessDestination('session_ended', data, slug) ?? '/login')
  })
}
