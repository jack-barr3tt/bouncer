import { realpathSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { changeTypeNames, releaseNotes } from './notes.mjs'
import { githubToken, headDate, pullsBetween, repository } from './pulls.mjs'

async function main() {
  const [fromArg, toArg, versionArg] = process.argv.slice(2)
  const to = toArg || 'HEAD'
  const from = !fromArg || fromArg === '-' ? '' : fromArg
  const version = (versionArg || '').replace(/^v/, '')
  if (!version) throw new Error('usage: release-notes.mjs <from|-> <to> <version>')
  const pulls = await pullsBetween(repository(), from, to, githubToken())
  const { notes, errors } = releaseNotes(pulls, version, headDate(to))
  if (errors.length > 0) {
    for (const error of errors) console.error(error)
    console.error(`add one of: ${changeTypeNames.join(', ')}`)
    process.exit(1)
  }
  process.stdout.write(notes)
}

if (process.argv[1] && realpathSync(process.argv[1]) === realpathSync(fileURLToPath(import.meta.url))) {
  main().catch((error) => {
    console.error(error.message)
    process.exit(1)
  })
}
