#!/usr/bin/env node
import { existsSync } from 'node:fs'
import { createInterface } from 'node:readline/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { parseArgs } from 'node:util'
import { ciSystems, writeSite, type CiSystem } from './write.js'

const usage = `Usage: create-bouncer [directory] --ci github|woodpecker|forgejo --public-url URL --site-path PATH

Writes the site CI pipelines, the build and deploy scripts, and deploy/docker-compose.yml.
Prompts for anything left out when stdin is a terminal.`

function assetsRoot(): string {
  const here = dirname(fileURLToPath(import.meta.url))
  const vendored = join(here, '..', 'vendor')
  if (existsSync(join(vendored, 'templates'))) return vendored
  return join(here, '..', '..')
}

function isCi(value: string): value is CiSystem {
  return (ciSystems as readonly string[]).includes(value)
}

async function main(): Promise<void> {
  const { values, positionals } = parseArgs({
    options: {
      ci: { type: 'string' },
      'public-url': { type: 'string' },
      'site-path': { type: 'string' },
      help: { type: 'boolean', short: 'h' },
    },
    allowPositionals: true,
  })
  if (values.help) {
    console.log(usage)
    return
  }

  const interactive = process.stdin.isTTY === true
  const rl = interactive ? createInterface({ input: process.stdin, output: process.stdout }) : null
  try {
    const dirArg = positionals[0] ?? (await prompt(rl, 'Directory', '.'))
    const ci = parseCi(values.ci ?? (await promptChoice(rl)))
    const publicUrl = requireUrl(values['public-url'] ?? (await prompt(rl, 'Public URL', 'https://apps.example.com')))
    const sitePath = requirePath(values['site-path'] ?? (await prompt(rl, 'Site directory on the server', '/var/www/apps')))
    const dir = resolve(dirArg)
    const { wrote, skipped } = writeSite({ dir, ci, publicUrl, sitePath, assetsRoot: assetsRoot() })
    printResult(dir, ci, publicUrl, wrote, skipped)
  } finally {
    rl?.close()
  }
}

async function prompt(rl: ReturnType<typeof createInterface> | null, label: string, fallback: string): Promise<string> {
  if (!rl) missing(label)
  const answer = (await rl.question(`${label} [${fallback}]: `)).trim()
  return answer || fallback
}

async function promptChoice(rl: ReturnType<typeof createInterface> | null): Promise<string> {
  if (!rl) missing('ci')
  console.log('CI system:')
  ciSystems.forEach((name, index) => {
    console.log(`  ${index + 1}) ${name}`)
  })
  const answer = (await rl.question('Choice [github]: ')).trim().toLowerCase()
  if (answer === '' || answer === '1') return 'github'
  if (answer === '2') return 'woodpecker'
  if (answer === '3') return 'forgejo'
  return answer
}

function parseCi(value: string): CiSystem {
  const name = value.trim().toLowerCase()
  if (!isCi(name)) {
    console.error(`ci must be one of: ${ciSystems.join(', ')}`)
    process.exit(1)
  }
  return name
}

function requireUrl(value: string): string {
  const url = value.trim().replace(/\/$/, '')
  if (!url.startsWith('http://') && !url.startsWith('https://')) {
    console.error('public URL must start with http:// or https://')
    process.exit(1)
  }
  return url
}

function requirePath(value: string): string {
  const path = value.trim()
  if (path === '' || !path.startsWith('/')) {
    console.error('site path must be an absolute directory')
    process.exit(1)
  }
  return path
}

function missing(flag: string): never {
  console.error(`${usage}\n\nMissing ${flag}.`)
  process.exit(1)
}

function printResult(dir: string, ci: CiSystem, publicUrl: string, wrote: string[], skipped: string[]): void {
  console.log(`\nSite files in ${dir}`)
  if (wrote.length > 0) {
    console.log('Wrote:')
    for (const file of wrote) console.log(`  ${file}`)
  }
  if (skipped.length > 0) {
    console.log('Left in place:')
    for (const file of skipped) console.log(`  ${file}`)
  }
  console.log('\nOn the server, copy deploy/ into place, copy deploy/.env.example to deploy/.env, and set the bootstrap password.')
  console.log('Put a read-only deploy key in deploy/git-key.')
  console.log('From that directory: docker compose up -d')
  console.log(`\nWebhook: ${publicUrl}/api/hooks/git`)
  console.log('Set DEPLOY_WEBHOOK_SECRET in deploy/.env and paste that secret into the forge.')
  console.log('\nCI secrets, when the pipeline should trigger the deploy:')
  if (ci === 'woodpecker') {
    console.log('  deploy_token')
    console.log('  deploy_url     public URL with no path')
  } else {
    console.log('  DEPLOY_TOKEN')
    console.log('  DEPLOY_URL      public URL with no path')
  }
}

const isMain = process.argv[1] !== undefined && import.meta.url === pathToFileURL(process.argv[1]).href
if (isMain) {
  main().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error)
    process.exit(1)
  })
}
