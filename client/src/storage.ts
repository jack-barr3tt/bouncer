export class StorageConflict extends Error {
  constructor() {
    super('The stored document changed. Load it again.')
    this.name = 'StorageConflict'
  }
}

export class Storage {
  private readonly slug: string
  private readonly path: string

  constructor(slug: string) {
    this.slug = slug
    this.path = `/api/apps/${encodeURIComponent(slug)}/storage`
  }

  getItem(key: string): Promise<string | null> {
    return enqueue(this.slug, async () => {
      const doc = await pull(this.path)
      if (!doc) return null
      return doc.data[String(key)] ?? null
    })
  }

  setItem(key: string, value: string): Promise<void> {
    const name = String(key)
    const stored = String(value)
    return enqueue(this.slug, () => change(this.path, (data) => {
      data[name] = stored
    }))
  }

  removeItem(key: string): Promise<void> {
    const name = String(key)
    return enqueue(this.slug, () => change(this.path, (data) => {
      delete data[name]
    }))
  }

  clear(): Promise<void> {
    return enqueue(this.slug, () => change(this.path, (data) => {
      for (const name of Object.keys(data)) delete data[name]
    }))
  }

  async key(index: number): Promise<string | null> {
    const names = await this.names()
    const at = Math.trunc(index)
    if (at < 0 || at >= names.length) return null
    return names[at] ?? null
  }

  async length(): Promise<number> {
    const names = await this.names()
    return names.length
  }

  private names(): Promise<string[]> {
    return enqueue(this.slug, async () => {
      const doc = await pull(this.path)
      if (!doc) return []
      return Object.keys(doc.data)
    })
  }
}

type Document = {
  data: Record<string, string>
  revision: string
}

const tails = new Map<string, Promise<unknown>>()

function enqueue<T>(slug: string, op: () => Promise<T>): Promise<T> {
  const prev = tails.get(slug) ?? Promise.resolve()
  const run = prev.then(op, op)
  tails.set(slug, run.then(() => undefined, () => undefined))
  return run
}

async function change(path: string, apply: (data: Record<string, string>) => void): Promise<void> {
  for (let attempt = 0; attempt < 5; attempt++) {
    const doc = await pull(path)
    if (!doc) throw new Error('Sign in required.')
    const next = { ...doc.data }
    apply(next)
    if (same(doc.data, next)) return
    try {
      await push(path, next, doc.revision)
      return
    } catch (error) {
      if (!(error instanceof StorageConflict)) throw error
    }
  }
  throw new StorageConflict()
}

function same(left: Record<string, string>, right: Record<string, string>): boolean {
  const leftKeys = Object.keys(left)
  const rightKeys = Object.keys(right)
  if (leftKeys.length !== rightKeys.length) return false
  return leftKeys.every((key) => left[key] === right[key])
}

async function pull(path: string): Promise<Document | null> {
  const response = await fetch(path, { credentials: 'include' })
  if (response.status === 401) return null
  if (!response.ok) throw new Error(await failure(response))
  const body = (await response.json()) as { data?: unknown; revision?: unknown }
  const data: Record<string, string> = {}
  if (body.data !== null && typeof body.data === 'object' && !Array.isArray(body.data)) {
    for (const [key, value] of Object.entries(body.data)) {
      if (typeof value === 'string') data[key] = value
    }
  }
  return { data, revision: typeof body.revision === 'string' ? body.revision : '' }
}

async function push(path: string, data: Record<string, string>, revision: string): Promise<void> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (revision !== '') headers['If-Match'] = revision
  const response = await fetch(path, {
    method: 'PUT',
    credentials: 'include',
    headers,
    body: JSON.stringify(data),
  })
  if (response.status === 409) throw new StorageConflict()
  if (response.status === 413) throw new DOMException('Document is too large.', 'QuotaExceededError')
  if (!response.ok) throw new Error(await failure(response))
}

async function failure(response: Response): Promise<string> {
  try {
    const body = (await response.json()) as { message?: unknown }
    if (typeof body.message === 'string' && body.message !== '') return body.message
  } catch {
    // The failure body is not JSON.
  }
  return `Storage request failed (${response.status}).`
}
