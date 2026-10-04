import { execFileSync } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { listAppSlugs, shouldBuildAll, slugsFromDiff } from './changed.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const apps = listAppSlugs(join(root, 'apps'))
const before = process.env.EVENT_BEFORE || process.env.CI_PREV_COMMIT_SHA || ''
const sha = process.env.EVENT_SHA || process.env.CI_COMMIT_SHA || 'HEAD'

let targets = apps
if (!shouldBuildAll(before)) {
  try {
    const diff = execFileSync('git', ['diff', '--name-only', before, sha], {
      cwd: root,
      encoding: 'utf8',
    })
    targets = slugsFromDiff(diff, apps)
  } catch {
    targets = apps
  }
}

if (targets.length === 0) {
  console.log('no apps to build')
  process.exit(0)
}

for (const slug of targets) {
  const dir = join(root, 'apps', slug)
  console.log(`building ${slug}`)
  execFileSync('npm', ['ci'], { cwd: dir, stdio: 'inherit' })
  execFileSync('npm', ['run', 'build'], { cwd: dir, stdio: 'inherit' })
}
