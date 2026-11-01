import { realpathSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { git, previousVersionTag } from './pulls.mjs'

export const releaseArtifacts = ['client', 'create', 'image', 'builder']

const watches = {
  client: (path) => path.startsWith('client/'),
  create: (path) =>
    path.startsWith('create/') ||
    path.startsWith('templates/') ||
    path.startsWith('scaffold/') ||
    path === 'deploy/docker-compose.yml',
  image: (path) =>
    path === 'Dockerfile' ||
    path === '.dockerignore' ||
    path.startsWith('server/') ||
    path.startsWith('hub/') ||
    path.startsWith('scaffold/') ||
    path === 'scripts/docker-entrypoint.sh',
  builder: (path) => path === 'Dockerfile' || path === '.dockerignore' || path.startsWith('server/'),
}

export function artifactsForDiff(paths) {
  return releaseArtifacts.filter((name) => paths.some((path) => watches[name](path)))
}

function changedPaths(from) {
  return git(['diff', '--name-only', from, 'HEAD'])
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

function main() {
  const from = previousVersionTag(process.env.CI_COMMIT_TAG || '')
  const names = from ? artifactsForDiff(changedPaths(from)) : releaseArtifacts
  process.stdout.write(names.join('\n') + (names.length ? '\n' : ''))
}

if (process.argv[1] && realpathSync(process.argv[1]) === realpathSync(fileURLToPath(import.meta.url))) {
  try {
    main()
  } catch (error) {
    console.error(error.message)
    process.exit(1)
  }
}
