const sections = [
  ['security', 'Security'],
  ['feature', 'Features'],
  ['fix', 'Fixes'],
  ['docs', 'Documentation'],
  ['test', 'Tests'],
  ['deps', 'Dependencies'],
  ['chore', 'Maintenance'],
]

const canonical = new Map(sections.map(([name]) => [name.toLowerCase(), name]))
const titles = Object.fromEntries(sections)

export const changeTypeNames = sections.map(([name]) => name)

export function splitLabels(value) {
  return value
    .split(',')
    .map((label) => label.trim())
    .filter(Boolean)
}

export function selectChangeType(labels) {
  const found = []
  for (const label of labels) {
    const name = canonical.get(label.toLowerCase())
    if (name && !found.includes(name)) found.push(name)
  }
  return found.length === 1 ? { type: found[0], found } : { type: '', found }
}

function linkText(title, number) {
  const text = title.replaceAll('[', '(').replaceAll(']', ')').replaceAll('\n', ' ').trim()
  return text || `#${number}`
}

export function releaseNotes(pulls, version, date) {
  const errors = []
  const groups = new Map()
  for (const pull of pulls) {
    const { type, found } = selectChangeType(pull.labels)
    if (!type) {
      const problem = found.length > 1 ? `has ${found.join(' and ')}` : 'has no change type'
      errors.push(`#${pull.number} ${problem}: ${pull.url}`)
      continue
    }
    const name = titles[type]
    if (!groups.has(name)) groups.set(name, [])
    groups.get(name).push(`- [${linkText(pull.title, pull.number)}](${pull.url})`)
  }
  if (errors.length > 0 || groups.size === 0) return { notes: '', errors }
  const used = sections.map(([, name]) => name).filter((name) => groups.has(name))
  const parts = [`## ${String(version).replace(/^v/, '')} - ${date}`, '']
  for (const name of used) {
    parts.push(`### ${name}`, ...groups.get(name), '')
  }
  return { notes: `${parts.join('\n').trim()}\n`, errors }
}
