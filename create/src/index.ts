#!/usr/bin/env node
import { existsSync, realpathSync } from 'node:fs'
import { createInterface } from 'node:readline/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { parseArgs } from 'node:util'
import { ciSystems, writeSite, type CiSystem } from './write.js'

const usage = `Usage: create-bouncer [directory] --ci github|woodpecker|forgejo --public-url URL --site-path PATH [--port PORT]

Writes the site CI pipelines, the build and deploy scripts, deploy/docker-compose.yml, and the Hello sample.
--port is the host port on 127.0.0.1. It defaults to 8080.
Prompts for anything left out when stdin is a terminal.`

function assetsRoot(): string {
  const here = dirname(fileURLToPath(import.meta.url))
  const vendored = join(here, '..', 'vendor')
  if (existsSync(join(vendored, 'templates')) && existsSync(join(vendored, 'scaffold', 'apps.yaml'))) return vendored
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
      port: { type: 'string' },
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
    if (interactive) {
      console.log('This writes a Bouncer site: apps.yaml, the Hello sample, CI, and deploy/.')
      console.log('Commit that directory, then run Bouncer on the server from its deploy/ folder.')
    }
    const dirArg = positionals[0] ?? (await prompt(rl, 'Where to write the site', '.', 'On this machine. Use . for the current directory.'))
    const ci = parseCi(values.ci ?? (await promptChoice(rl)))
    const publicUrl = requireUrl(values['public-url'] ?? (await prompt(rl, 'Public URL', 'https://apps.example.com', 'The address people open. Put TLS on a proxy in front of Bouncer.')))
    const sitePath = requirePath(values['site-path'] ?? (await prompt(rl, 'Path of this directory on the server', '/var/www/apps', 'Where you will put this site on the server. Docker mounts that path. It is saved as SITE_PATH.')))
    const port = requirePort(values.port ?? (rl ? await prompt(rl, 'Host port on the server', '8080', 'Published on 127.0.0.1. Point your proxy at this port. Saved as PORT.') : '8080'))
    const dir = resolve(dirArg)
    const { wrote, skipped } = writeSite({ dir, ci, publicUrl, sitePath, port, assetsRoot: assetsRoot() })
    printResult(dir, sitePath, port, ci, publicUrl, wrote, skipped)
  } finally {
    rl?.close()
  }
}

async function prompt(rl: ReturnType<typeof createInterface> | null, label: string, fallback: string, hint: string): Promise<string> {
  if (!rl) missing(label)
  console.log(`\n${hint}`)
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
  let path = value.trim()
  if (path.length > 1) path = path.replace(/\/+$/, '')
  if (path === '' || !path.startsWith('/')) {
    console.error('site path must be an absolute directory')
    process.exit(1)
  }
  return path
}

function requirePort(value: string): string {
  if (!/^[0-9]+$/.test(value.trim())) {
    console.error('port must be a number from 1 to 65535')
    process.exit(1)
  }
  const port = Number(value)
  if (port < 1 || port > 65535) {
    console.error('port must be a number from 1 to 65535')
    process.exit(1)
  }
  return String(port)
}

function missing(flag: string): never {
  console.error(`${usage}\n\nMissing ${flag}.`)
  process.exit(1)
}

function printResult(dir: string, sitePath: string, port: string, ci: CiSystem, publicUrl: string, wrote: string[], skipped: string[]): void {
  console.log(`\nSite files in ${dir}`)
  if (wrote.length > 0) {
    console.log('Wrote:')
    for (const file of wrote) console.log(`  ${file}`)
  }
  if (skipped.length > 0) {
    console.log('Left in place:')
    for (const file of skipped) console.log(`  ${file}`)
  }
  const host = ciHost(ci)
  console.log('')
  if (dir === sitePath) {
    console.log(`This directory is already at ${sitePath} on the server.`)
  } else {
    console.log(`Clone this repository to ${sitePath} on the server.`)
  }
  console.log('deploy/ is already in the repository. It runs Bouncer. From there:')
  console.log(`  cd ${sitePath}/deploy`)
  console.log('  cp .env.example .env')
  console.log('  # set AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD')
  console.log('  docker compose up -d')
  console.log('')
  console.log(`Bouncer listens on 127.0.0.1:${port}. Point TLS and your hostname at that port.`)
  console.log('Change PORT in deploy/.env if that port is taken.')
  console.log('Leave GIT_REMOTE empty in deploy/.env to serve the files already in the clone.')
  console.log('')
  console.log('To publish a push, set GIT_REMOTE in deploy/.env, replace deploy/git-key with a read-only SSH key for the repository, and set BUILDER_TOKEN to a long secret. Then add a webhook, the CI secrets, or both.')
  console.log('')
  console.log(`Webhook. In ${host}, add a webhook for push events to:`)
  console.log(`  ${publicUrl}/api/hooks/git`)
  console.log('Choose a long secret. Set DEPLOY_WEBHOOK_SECRET in deploy/.env to that secret, and enter the same secret in the webhook.')
  console.log('')
  console.log(`CI secrets. The deploy workflow in this repository calls Bouncer on a push to main. In the ${host} repository secrets, set:`)
  if (ci === 'woodpecker') {
    console.log('  deploy_token    same value as DEPLOY_TOKEN in deploy/.env')
    console.log(`  deploy_url      ${publicUrl}`)
  } else {
    console.log('  DEPLOY_TOKEN    same value as DEPLOY_TOKEN in deploy/.env')
    console.log(`  DEPLOY_URL      ${publicUrl}`)
  }
}

function ciHost(ci: CiSystem): string {
  if (ci === 'woodpecker') return 'Woodpecker'
  if (ci === 'forgejo') return 'Forgejo'
  return 'GitHub'
}

const isMain = process.argv[1] !== undefined && realpathSync(process.argv[1]) === realpathSync(fileURLToPath(import.meta.url))
if (isMain) {
  main().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : error)
    process.exit(1)
  })
}
