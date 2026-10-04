import { cpSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const repo = join(here, '..')
const vendor = join(here, 'vendor')
rmSync(vendor, { recursive: true, force: true })
mkdirSync(join(vendor, 'deploy'), { recursive: true })
cpSync(join(repo, 'templates'), join(vendor, 'templates'), { recursive: true })
cpSync(join(repo, 'deploy', 'docker-compose.yml'), join(vendor, 'deploy', 'docker-compose.yml'))
