import YAML from 'yaml'

export type AppEntry = {
  slug: string
  name: string
  description: string
  path: string
  icon: string
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
