import { cpSync, mkdirSync, rmSync } from 'node:fs'
import { basename, join } from 'node:path'

const here = import.meta.dirname
const repo = join(here, '..')
const vendor = join(here, 'vendor')
rmSync(vendor, { recursive: true, force: true })
mkdirSync(join(vendor, 'deploy'), { recursive: true })
cpSync(join(repo, 'templates'), join(vendor, 'templates'), { recursive: true })
cpSync(join(repo, 'deploy', 'docker-compose.yml'), join(vendor, 'deploy', 'docker-compose.yml'))
cpSync(join(repo, 'deploy', 'env.yaml'), join(vendor, 'deploy', 'env.yaml'))
cpSync(join(repo, 'scaffold'), join(vendor, 'scaffold'), {
  recursive: true,
  filter: (src) => {
    const name = basename(src)
    return name !== 'node_modules' && name !== 'dist'
  },
})
