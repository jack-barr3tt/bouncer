import { chmodSync, copyFileSync, existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'

export const ciSystems = ['github', 'woodpecker', 'forgejo'] as const

export type CiSystem = (typeof ciSystems)[number]

export type SetupOptions = {
  dir: string
  ci: CiSystem
  publicUrl: string
  sitePath: string
  port: string
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

const gitignoreLines = ['node_modules/', 'dist/', 'deploy/.env', 'deploy/git-key', 'deploy/update.json']

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
  writeKey(options, wrote, skipped)
  writeGitignore(options.dir, wrote, skipped)
  copySample(options, wrote, skipped)
  return { wrote, skipped }
}

const sampleSkip = new Set(['node_modules', 'dist'])

function copySample(options: SetupOptions, wrote: string[], skipped: string[]): void {
  copyNew(options, join('scaffold', 'AGENTS.md'), 'AGENTS.md', wrote, skipped)
  copyNew(options, join('scaffold', 'apps.AGENTS.md'), join('apps', 'AGENTS.md'), wrote, skipped)
  copyNew(options, join('scaffold', 'apps.yaml'), 'apps.yaml', wrote, skipped)

  const to = join('apps', 'hello')
  const dest = join(options.dir, to)
  if (existsSync(dest)) {
    skipped.push(to)
    return
  }
  copySampleFiles(join(options.assetsRoot, 'scaffold', 'apps', 'hello'), dest, to, wrote)
}

function copySampleFiles(src: string, dest: string, rel: string, wrote: string[]): void {
  mkdirSync(dest, { recursive: true })
  for (const entry of readdirSync(src, { withFileTypes: true })) {
    if (sampleSkip.has(entry.name)) continue
    const from = join(src, entry.name)
    const target = join(dest, entry.name)
    const name = join(rel, entry.name)
    if (entry.isDirectory()) {
      copySampleFiles(from, target, name, wrote)
      continue
    }
    if (!entry.isFile()) continue
    copyFileSync(from, target)
    wrote.push(name)
  }
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
  const fields = parseEnvSpec(readFileSync(join(options.assetsRoot, 'deploy', 'env.yaml'), 'utf8'))
  setField(fields, 'PORT', options.port)
  setField(fields, 'SITE_PATH', options.sitePath)
  setField(fields, 'PUBLIC_BASE_URL', options.publicUrl)
  setField(fields, 'COOKIE_SECURE', secure ? 'true' : 'false')
  mkdirSync(dirname(dest), { recursive: true })
  writeFileSync(dest, renderEnv(fields))
  wrote.push(to)
}

type EnvField = { name: string; value: string; required: boolean }

function parseEnvSpec(text: string): EnvField[] {
  const fields: EnvField[] = []
  let current: EnvField | null = null
  for (const raw of text.split('\n')) {
    const line = raw.trim()
    if (line === '' || line.startsWith('#')) continue
    const item = line.match(/^- name:\s*([A-Za-z0-9_]+)\s*$/)
    if (item) {
      if (current) fields.push(current)
      current = { name: item[1], value: '', required: false }
      continue
    }
    if (!current) throw new Error(`env spec: ${line}`)
    const value = line.match(/^value:\s*(.*)$/)
    if (value) {
      current.value = unquote(value[1].trim())
      continue
    }
    if (line === 'required: true') {
      current.required = true
      continue
    }
    throw new Error(`env spec: ${line}`)
  }
  if (current) fields.push(current)
  return fields
}

function unquote(value: string): string {
  if (value.length >= 2 && value.startsWith('"') && value.endsWith('"')) return value.slice(1, -1)
  return value
}

function setField(fields: EnvField[], name: string, value: string): void {
  const field = fields.find((item) => item.name === name)
  if (!field) throw new Error(`env spec: missing ${name}`)
  field.value = value
  field.required = false
}

function renderEnv(fields: EnvField[]): string {
  return `${fields.map((field) => `${field.name}=${field.value}`).join('\n')}\n`
}

function writeKey(options: SetupOptions, wrote: string[], skipped: string[]): void {
  const to = join('deploy', 'git-key')
  const dest = join(options.dir, to)
  if (existsSync(dest)) {
    skipped.push(to)
    return
  }
  mkdirSync(dirname(dest), { recursive: true })
  writeFileSync(dest, '')
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
