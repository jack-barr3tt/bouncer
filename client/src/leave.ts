export function accessDestination(
  name: string,
  data: { apps?: unknown; hub?: unknown },
  slug: string,
): string | null {
  const hub = typeof data.hub === 'string' ? data.hub : ''
  if (name === 'session_ended') return loginURL(hub)
  if (name !== 'access') return null
  const apps = Array.isArray(data.apps) ? data.apps : []
  if (!apps.includes(slug)) return homeURL(hub)
  return null
}

function homeURL(hub: string): string {
  const base = hub.replace(/\/+$/, '')
  return base === '' ? '/' : `${base}/`
}

function loginURL(hub: string): string {
  const base = hub.replace(/\/+$/, '')
  return base === '' ? '/login' : `${base}/login`
}
