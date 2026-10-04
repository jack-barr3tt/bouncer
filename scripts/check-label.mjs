import { changeTypeNames, selectChangeType, splitLabels } from './notes.mjs'

const labels = splitLabels(process.env.CI_COMMIT_PULL_REQUEST_LABELS || '')
const { type, found } = selectChangeType(labels)
if (type) {
  console.log(type)
  process.exit(0)
}
if (found.length > 1) console.error(`pull request has ${found.join(' and ')}`)
else console.error('pull request has no change type')
console.error(`add one of: ${changeTypeNames.join(', ')}`)
process.exit(1)
