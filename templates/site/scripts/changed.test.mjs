import assert from 'node:assert/strict'
import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { test } from 'node:test'
import { listAppSlugs, shouldBuildAll, slugsFromDiff } from './changed.mjs'

test('a missing base builds every app', () => {
  assert.equal(shouldBuildAll(''), true)
  assert.equal(shouldBuildAll('0000000'), true)
  assert.equal(shouldBuildAll('abc123'), false)
})

test('a diff selects known app directories', () => {
  const diff = ['apps/hello/src/App.tsx', 'apps/hub/index.html', 'apps.yaml', 'README.md'].join('\n')
  assert.deepEqual(slugsFromDiff(diff, ['hello', 'notes']), ['hello'])
})

test('listAppSlugs skips reserved names and folders without a package', () => {
  const dir = mkdtempSync(join(tmpdir(), 'apps-'))
  mkdirSync(join(dir, 'hello'))
  writeFileSync(join(dir, 'hello', 'package.json'), '{}')
  mkdirSync(join(dir, 'hub'))
  writeFileSync(join(dir, 'hub', 'package.json'), '{}')
  mkdirSync(join(dir, 'notes'))
  assert.deepEqual(listAppSlugs(dir), ['hello'])
})
