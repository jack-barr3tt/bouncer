import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { chmodSync, mkdtempSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { test } from 'node:test'

const entrypoint = join(import.meta.dirname, 'docker-entrypoint.sh')

function scaffold(root) {
  const dir = join(root, 'scaffold')
  mkdirSync(join(dir, 'apps', 'hello', 'dist'), { recursive: true })
  writeFileSync(join(dir, 'AGENTS.md'), 'agents\n')
  writeFileSync(join(dir, 'apps.AGENTS.md'), 'apps agents\n')
  writeFileSync(join(dir, 'apps.yaml'), 'apps:\n  - slug: hello\n')
  writeFileSync(join(dir, 'apps', 'hello', 'package.json'), '{}\n')
  writeFileSync(join(dir, 'apps', 'hello', 'dist', 'index.html'), 'built\n')
  return dir
}

function init(site, scaffoldDir) {
  const bin = join(site, '..', 'bin')
  const log = join(site, '..', 'bouncer.log')
  mkdirSync(bin)
  writeFileSync(join(bin, 'bouncer'), '#!/bin/sh\nprintf "%s\\n" "$*" >> "$BOUNCER_LOG"\nexit 0\n')
  chmodSync(join(bin, 'bouncer'), 0o755)
  const result = spawnSync('sh', [entrypoint, 'init'], {
    env: { ...process.env, PATH: `${bin}:${process.env.PATH}`, SITE_ROOT: site, SCAFFOLD: scaffoldDir, BOUNCER_LOG: log },
    encoding: 'utf8',
  })
  result.log = readFileSync(log, 'utf8')
  return result
}

test('init writes Hello into an empty site', () => {
  const root = mkdtempSync(join(tmpdir(), 'bouncer-entry-'))
  const site = join(root, 'site')
  const result = init(site, scaffold(root))
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.log, /^adopt\n/)
  assert.equal(readFileSync(join(site, 'apps', 'hello', 'dist', 'index.html'), 'utf8'), 'built\n')
  assert.match(readFileSync(join(site, 'apps.yaml'), 'utf8'), /slug: hello/)
})

test('init fills a missing Hello build and leaves apps.yaml in place', () => {
  const root = mkdtempSync(join(tmpdir(), 'bouncer-entry-'))
  const site = join(root, 'site')
  mkdirSync(join(site, 'apps', 'hello'), { recursive: true })
  writeFileSync(join(site, 'apps.yaml'), 'kept\n')
  writeFileSync(join(site, 'apps', 'hello', 'package.json'), '{}\n')
  const result = init(site, scaffold(root))
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.log, /^adopt\n/)
  assert.equal(readFileSync(join(site, 'apps.yaml'), 'utf8'), 'kept\n')
  assert.equal(readFileSync(join(site, 'apps', 'hello', 'dist', 'index.html'), 'utf8'), 'built\n')
})
