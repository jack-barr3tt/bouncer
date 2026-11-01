import assert from 'node:assert/strict'
import { afterEach, test } from 'node:test'
import { Storage, StorageConflict } from '../src/storage.ts'

const realFetch = globalThis.fetch

afterEach(() => {
  globalThis.fetch = realFetch
})

test('getItem returns null when signed out', async () => {
  globalThis.fetch = async () => Response.json({ message: 'Sign in required.' }, { status: 401 })
  assert.equal(await new Storage('hello').getItem('theme'), null)
})

test('setItem and getItem use string keys', async () => {
  const remote = document()
  globalThis.fetch = remote.fetch
  const storage = new Storage('hello')
  await storage.setItem('theme', 'dark')
  await storage.setItem('name', 'Ada')
  assert.equal(await storage.getItem('theme'), 'dark')
  assert.equal(await storage.getItem('name'), 'Ada')
  assert.equal(await storage.length(), 2)
  assert.equal(await storage.key(0), 'theme')
  assert.equal(await storage.key(2), null)
  assert.equal(remote.puts[0]?.match, null)
  assert.equal(remote.puts[1]?.match, '1')
  assert.deepEqual(remote.puts[1]?.body, { theme: 'dark', name: 'Ada' })
})

test('removeItem and clear update the document', async () => {
  const remote = document({ theme: 'dark', name: 'Ada' }, '4')
  globalThis.fetch = remote.fetch
  const storage = new Storage('notes')
  await storage.removeItem('theme')
  assert.deepEqual(remote.puts.at(-1)?.body, { name: 'Ada' })
  await storage.clear()
  assert.deepEqual(remote.puts.at(-1)?.body, {})
  assert.equal(await storage.getItem('name'), null)
})

test('a conflict retries with the other writer kept', async () => {
  let puts = 0
  let data: Record<string, string> = {}
  let revision = ''
  globalThis.fetch = async (_input, init) => {
    if ((init?.method ?? 'GET') === 'GET') return Response.json({ data, revision })
    puts++
    if (puts === 1) {
      data = { other: 'kept' }
      revision = '9'
      return Response.json({ message: 'conflict' }, { status: 409 })
    }
    data = JSON.parse(String(init?.body)) as Record<string, string>
    revision = '10'
    return Response.json({ revision })
  }
  await new Storage('hello').setItem('theme', 'dark')
  assert.deepEqual(data, { other: 'kept', theme: 'dark' })
})

test('a document that is too large raises QuotaExceededError', async () => {
  globalThis.fetch = async (_input, init) => {
    if ((init?.method ?? 'GET') === 'GET') return Response.json({ data: {}, revision: '' })
    return Response.json({ message: 'Document is too large.' }, { status: 413 })
  }
  await assert.rejects(() => new Storage('hello').setItem('theme', 'dark'), (error: unknown) => {
    assert.ok(error instanceof DOMException)
    assert.equal(error.name, 'QuotaExceededError')
    return true
  })
})

test('retries give up when the document keeps changing', async () => {
  globalThis.fetch = async (_input, init) => {
    if ((init?.method ?? 'GET') === 'GET') return Response.json({ data: {}, revision: '' })
    return Response.json({ message: 'conflict' }, { status: 409 })
  }
  await assert.rejects(() => new Storage('hello').setItem('theme', 'dark'), StorageConflict)
})

function document(initial: Record<string, string> = {}, initialRevision = '') {
  let data = { ...initial }
  let revision = initialRevision
  let seq = initialRevision === '' ? 0 : Number(initialRevision)
  const puts: { match: string | null; body: Record<string, string> }[] = []
  return {
    puts,
    fetch: async (_input: unknown, init?: RequestInit) => {
      if ((init?.method ?? 'GET') === 'GET') return Response.json({ data, revision })
      const match = new Headers(init?.headers).get('If-Match')
      const body = JSON.parse(String(init?.body)) as Record<string, string>
      puts.push({ match, body })
      if ((revision === '' && match !== null) || (revision !== '' && match !== revision)) {
        return Response.json({ message: 'conflict' }, { status: 409 })
      }
      data = body
      seq += 1
      revision = String(seq)
      return Response.json({ revision })
    },
  }
}
