import { existsSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const reserved = new Set(['hub', 'assets', 'code'])

export function shouldBuildAll(before) {
  return !before || /^0+$/.test(before)
}

export function listAppSlugs(appsDir) {
  if (!existsSync(appsDir)) return []
  return readdirSync(appsDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && !entry.name.startsWith('.') && !reserved.has(entry.name))
    .map((entry) => entry.name)
    .filter((name) => existsSync(join(appsDir, name, 'package.json')))
    .sort()
}

export function slugsFromDiff(diff, apps) {
  const known = new Set(apps)
  const slugs = new Set()
  for (const line of diff.split('\n')) {
    const match = /^apps\/([^/]+)\//.exec(line)
    if (match && known.has(match[1])) slugs.add(match[1])
  }
  return [...slugs].sort()
}
