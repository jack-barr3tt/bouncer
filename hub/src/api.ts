import YAML from 'yaml'

export type AppEntry = {
  slug: string
  name: string
  description: string
  path: string
  icon: string
}

export type Session = {
  kind: 'user' | 'temporary'
  username?: string
  nickname?: string
  role?: 'admin' | 'user'
  apps: string[]
  expiresAt: string
}

export type Account = {
  id: string
  username: string
  role: 'admin' | 'user'
  disabled: boolean
  apps: string[]
}

export type AccessCode = {
  id: string
  code: string
  label: string
  url: string
  expiresAt: string
  maxSignups: number
  signupCount: number
  revokedAt?: string
  apps: string[]
  createdAt: string
}

export type TemporaryAccount = {
  id: string
  nickname: string
  createdIp: string
  lastIp: string
  createdAt: string
  revokedAt?: string
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function send<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  const response = await fetch(path, { ...init, headers, credentials: 'include' })
  if (response.status === 204) return undefined as T
  const text = await response.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = null
    }
  }
  if (!response.ok) {
    const message =
      typeof data === 'object' && data !== null && 'message' in data && typeof data.message === 'string'
        ? data.message
        : `Request failed (${response.status}).`
    throw new ApiError(response.status, message)
  }
  return data as T
}

export function getSession(): Promise<Session | null> {
  return send<Session>('/api/session').catch((error: unknown) => {
    if (error instanceof ApiError && error.status === 401) return null
    throw error
  })
}

export function login(username: string, password: string, fingerprint: string): Promise<Session> {
  return send('/api/login', {
    method: 'POST',
    body: JSON.stringify({ username, password, fingerprint }),
  })
}

export function redeem(code: string, nickname: string, fingerprint: string): Promise<Session> {
  return send('/api/access-codes/redeem', {
    method: 'POST',
    body: JSON.stringify({ code, nickname, fingerprint }),
  })
}

export function logout(): Promise<void> {
  return send('/api/logout', { method: 'POST' })
}

export function listUsers(): Promise<{ users: Account[] }> {
  return send('/api/users')
}

export function createUser(username: string, password: string): Promise<Account> {
  return send('/api/users', { method: 'POST', body: JSON.stringify({ username, password }) })
}

export function updateUser(
  id: string,
  body: { password?: string; disabled?: boolean; role?: 'admin' | 'user' },
): Promise<Account> {
  return send(`/api/users/${id}`, { method: 'PATCH', body: JSON.stringify(body) })
}

export function setUserApps(id: string, slugs: string[]): Promise<Account> {
  return send(`/api/users/${id}/apps`, { method: 'PUT', body: JSON.stringify({ slugs }) })
}

export function listCodes(): Promise<{ codes: AccessCode[] }> {
  return send('/api/access-codes')
}

export function createCode(body: {
  label?: string
  appSlugs: string[]
  expiresAt: string
  maxSignups: number
}): Promise<AccessCode> {
  return send('/api/access-codes', { method: 'POST', body: JSON.stringify(body) })
}

export function revokeCode(id: string): Promise<AccessCode> {
  return send(`/api/access-codes/${id}/revoke`, { method: 'POST' })
}

export function listTemporaryAccounts(id: string): Promise<{ accounts: TemporaryAccount[] }> {
  return send(`/api/access-codes/${id}/accounts`)
}

export function revokeTemporaryAccount(id: string): Promise<void> {
  return send(`/api/temporary-accounts/${id}/revoke`, { method: 'POST' })
}

export function isAppEntry(value: unknown): value is AppEntry {
  if (typeof value !== 'object' || value === null) return false
  const entry = value as Record<string, unknown>
  return (
    typeof entry.slug === 'string' &&
    entry.slug !== '' &&
    typeof entry.name === 'string' &&
    entry.name !== '' &&
    typeof entry.description === 'string' &&
    entry.description !== '' &&
    typeof entry.path === 'string' &&
    entry.path !== '' &&
    typeof entry.icon === 'string' &&
    entry.icon !== ''
  )
}

export async function loadRegistry(): Promise<AppEntry[]> {
  const response = await fetch(`${import.meta.env.BASE_URL}apps.yaml`, { cache: 'no-store' })
  if (!response.ok) throw new Error(`The app list responded with ${response.status}.`)
  const data: unknown = YAML.parse(await response.text())
  if (typeof data !== 'object' || data === null || !('apps' in data) || !Array.isArray(data.apps)) {
    throw new Error('The app list is missing an "apps" array.')
  }
  const valid = data.apps.filter(isAppEntry)
  if (data.apps.length > 0 && valid.length === 0) {
    throw new Error('The app list did not contain any valid applications.')
  }
  return valid
}

export function watchAccess(onAccess: (apps: string[]) => void, onEnded: () => void): () => void {
  const stream = new EventSource('/api/access/stream')
  stream.addEventListener('access', (event) => {
    const data = JSON.parse((event as MessageEvent).data) as { apps?: string[] }
    onAccess(data.apps ?? [])
  })
  stream.addEventListener('session_ended', () => {
    onEnded()
  })
  return () => stream.close()
}
