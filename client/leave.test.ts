import assert from 'node:assert/strict'
import { test } from 'node:test'
import { accessDestination } from './src/leave.ts'

test('a revoked app returns to the hub', () => {
  const dest = accessDestination('access', { apps: ['other'], hub: 'https://example.com' }, 'notes')
  assert.equal(dest, 'https://example.com/')
})

test('a revoked app stays on this origin when the event has no hub', () => {
  const dest = accessDestination('access', { apps: [] }, 'notes')
  assert.equal(dest, '/')
})

test('an access event for this app does not navigate', () => {
  const dest = accessDestination('access', { apps: ['notes'], hub: 'https://example.com/' }, 'notes')
  assert.equal(dest, null)
})

test('a session end goes to the hub login', () => {
  const dest = accessDestination('session_ended', { hub: 'https://example.com/' }, 'notes')
  assert.equal(dest, 'https://example.com/login')
})

test('a session end stays on this origin when the event has no hub', () => {
  const dest = accessDestination('session_ended', {}, 'notes')
  assert.equal(dest, '/login')
})
