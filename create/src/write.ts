import { chmodSync, copyFileSync, existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'

export const ciSystems = ['github', 'woodpecker', 'forgejo'] as const

export type CiSystem = (typeof ciSystems)[number]

export type SetupOptions = {
  dir: string
  ci: CiSystem
  publicUrl: string
  sitePath: string
  assetsRoot: string
}

const ciFiles: Record<CiSystem, [string, string][]> = {
  github: [
    ['templates/github/pr.yaml', '.github/workflows/pr.yaml'],
    ['templates/github/deploy.yaml', '.github/workflows/deploy.yaml'],
  ],
  woodpecker: [
    ['templates/woodpecker/pr.yaml', '.woodpecker/pr.yaml'],
    ['templates/woodpecker/deploy.yaml', '.woodpecker/deploy.yaml'],
  ],
  forgejo: [
    ['templates/forgejo/pr.yaml', '.forgejo/workflows/pr.yaml'],
    ['templates/forgejo/deploy.yaml', '.forgejo/workflows/deploy.yaml'],
  ],
}

const gitignoreLines = ['node_modules/', 'dist/', 'deploy/.env']

export function writeSite(options: SetupOptions): { wrote: string[]; skipped: string[] } {
  const wrote: string[] = []
  const skipped: string[] = []
  mkdirSync(options.dir, { recursive: true })

  for (const [from, to] of ciFiles[options.ci]) {
    copyNew(options, from, to, wrote, skipped)
  }

  const scriptsDir = join(options.assetsRoot, 'templates', 'site', 'scripts')
  for (const name of readdirSync(scriptsDir)) {
    const dest = join('scripts', name)
    if (copyNew(options, join('templates', 'site', 'scripts', name), dest, wrote, skipped) && name.endsWith('.sh')) {
      chmodSync(join(options.dir, dest), 0o755)
    }
  }

  copyNew(options, join('deploy', 'docker-compose.yml'), join('deploy', 'docker-compose.yml'), wrote, skipped)
  writeEnv(options, wrote, skipped)
  writeGitignore(options.dir, wrote, skipped)
  return { wrote, skipped }
}

function copyNew(
  options: SetupOptions,
  from: string,
  to: string,
  wrote: string[],
  skipped: string[],
): boolean {
  const dest = join(options.dir, to)
  if (existsSync(dest)) {
    skipped.push(to)
    return false
  }
  mkdirSync(dirname(dest), { recursive: true })
  copyFileSync(join(options.assetsRoot, from), dest)
  wrote.push(to)
  return true
}

function writeEnv(options: SetupOptions, wrote: string[], skipped: string[]): void {
  const to = join('deploy', '.env.example')
  const dest = join(options.dir, to)
  if (existsSync(dest)) {
    skipped.push(to)
    return
  }
  const secure = options.publicUrl.startsWith('https://')
  const body = [
    'REGISTRY=ghcr.io/jack-barr3tt',
    'IMAGE_TAG=latest',
    'PORT=8080',
    `SITE_PATH=${options.sitePath}`,
    `PUBLIC_BASE_URL=${options.publicUrl}`,
    'AUTH_BOOTSTRAP_USERNAME=',
    'AUTH_BOOTSTRAP_PASSWORD=',
    'TRUSTED_PROXIES=127.0.0.1',
    `COOKIE_SECURE=${secure ? 'true' : 'false'}`,
    '',
  ].join('\n')
  mkdirSync(dirname(dest), { recursive: true })
  writeFileSync(dest, body)
  wrote.push(to)
}

function writeGitignore(dir: string, wrote: string[], skipped: string[]): void {
  const dest = join(dir, '.gitignore')
  if (!existsSync(dest)) {
    writeFileSync(dest, `${gitignoreLines.join('\n')}\n`)
    wrote.push('.gitignore')
    return
  }
  const current = readFileSync(dest, 'utf8')
  const missing = gitignoreLines.filter((line) => !current.split('\n').includes(line))
  if (missing.length === 0) {
    skipped.push('.gitignore')
    return
  }
  const suffix = current.endsWith('\n') || current.length === 0 ? '' : '\n'
  writeFileSync(dest, `${current}${suffix}${missing.join('\n')}\n`)
  wrote.push('.gitignore')
}
