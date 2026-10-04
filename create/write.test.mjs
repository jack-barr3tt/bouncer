import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { mkdtempSync, readFileSync, symlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { test } from 'node:test'
import { writeSite } from './dist/write.js'

const assetsRoot = join(import.meta.dirname, '..')

function options(dir, ci) {
  return {
    dir,
    ci,
    publicUrl: 'https://apps.example.com',
    sitePath: '/var/www/apps',
    assetsRoot,
  }
}

test('github writes pipelines, scripts, and compose', () => {
  const dir = mkdtempSync(join(tmpdir(), 'bouncer-site-'))
  const { wrote, skipped } = writeSite(options(dir, 'github'))
  assert.deepEqual(skipped, [])
  assert.ok(wrote.includes('.github/workflows/deploy.yaml'))
  assert.ok(wrote.includes('scripts/ci-build.mjs'))
  assert.ok(wrote.includes('deploy/docker-compose.yml'))
  const workflow = readFileSync(join(dir, '.github', 'workflows', 'deploy.yaml'), 'utf8')
  assert.match(workflow, /DEPLOY_TOKEN/)
  const env = readFileSync(join(dir, 'deploy', '.env.example'), 'utf8')
  assert.match(env, /PUBLIC_BASE_URL=https:\/\/apps\.example\.com/)
  assert.match(env, /SITE_PATH=\/var\/www\/apps/)
  assert.match(env, /COOKIE_SECURE=true/)
  assert.match(env, /GIT_REMOTE=/)
  assert.match(env, /BUILDER_TOKEN=/)
})

test('an existing file is left in place', () => {
  const dir = mkdtempSync(join(tmpdir(), 'bouncer-site-'))
  writeFileSync(join(dir, '.gitignore'), 'already\n')
  writeSite(options(dir, 'woodpecker'))
  const again = writeSite(options(dir, 'woodpecker'))
  assert.ok(again.skipped.includes('.woodpecker/deploy.yaml'))
  assert.ok(again.skipped.includes('.gitignore'))
  assert.match(readFileSync(join(dir, '.gitignore'), 'utf8'), /^already\n/)
  assert.match(readFileSync(join(dir, '.gitignore'), 'utf8'), /deploy\/\.env/)
})

test('forgejo and the cli use the same layout', () => {
  const dir = mkdtempSync(join(tmpdir(), 'bouncer-site-'))
  const result = spawnSync(
    process.execPath,
    [
      join(import.meta.dirname, 'dist', 'index.js'),
      dir,
      '--ci',
      'forgejo',
      '--public-url',
      'http://127.0.0.1:8080',
      '--site-path',
      '/srv/apps',
    ],
    { encoding: 'utf8' },
  )
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stdout, /api\/hooks\/git/)
  assert.match(readFileSync(join(dir, '.forgejo', 'workflows', 'pr.yaml'), 'utf8'), /pull_request/)
  assert.match(readFileSync(join(dir, 'deploy', '.env.example'), 'utf8'), /COOKIE_SECURE=false/)
})

test('the cli runs when invoked through a symlink', () => {
  const dir = mkdtempSync(join(tmpdir(), 'bouncer-site-'))
  const link = join(dir, 'create-bouncer')
  symlinkSync(join(import.meta.dirname, 'dist', 'index.js'), link)
  const result = spawnSync(process.execPath, [link, '--help'], { encoding: 'utf8' })
  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stdout, /create-bouncer/)
})
