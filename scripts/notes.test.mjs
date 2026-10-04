import assert from 'node:assert/strict'
import { test } from 'node:test'
import { releaseNotes, selectChangeType, splitLabels } from './notes.mjs'

test('one change-type label selects the section', () => {
  assert.deepEqual(selectChangeType(splitLabels('ci, feature')), { type: 'feature', found: ['feature'] })
  assert.deepEqual(selectChangeType(['Security']), { type: 'security', found: ['security'] })
  assert.equal(selectChangeType(['ci']).type, '')
  assert.deepEqual(selectChangeType(['feature', 'fix']).found, ['feature', 'fix'])
})

test('release notes link pull requests under their change type', () => {
  const { notes, errors } = releaseNotes(
    [
      {
        number: 4,
        title: 'Stage npm packages',
        url: 'https://github.com/jack-barr3tt/bouncer/pull/4',
        labels: ['feature', 'ci'],
      },
      {
        number: 5,
        title: 'Drop the stale session',
        url: 'https://github.com/jack-barr3tt/bouncer/pull/5',
        labels: ['fix'],
      },
      {
        number: 6,
        title: 'Stop an open redirect',
        url: 'https://github.com/jack-barr3tt/bouncer/pull/6',
        labels: ['Security'],
      },
    ],
    '0.2.0',
    '2026-10-04',
  )
  assert.deepEqual(errors, [])
  assert.equal(
    notes,
    [
      '## 0.2.0 - 2026-10-04',
      '',
      '### Security',
      '- [Stop an open redirect](https://github.com/jack-barr3tt/bouncer/pull/6)',
      '',
      '### Features',
      '- [Stage npm packages](https://github.com/jack-barr3tt/bouncer/pull/4)',
      '',
      '### Fixes',
      '- [Drop the stale session](https://github.com/jack-barr3tt/bouncer/pull/5)',
      '',
    ].join('\n'),
  )
})

test('a pull request needs exactly one change type', () => {
  const { notes, errors } = releaseNotes(
    [
      { number: 7, title: 'Clean up', url: 'https://github.com/jack-barr3tt/bouncer/pull/7', labels: [] },
      {
        number: 8,
        title: 'Both',
        url: 'https://github.com/jack-barr3tt/bouncer/pull/8',
        labels: ['feature', 'fix'],
      },
    ],
    '0.2.0',
    '2026-10-04',
  )
  assert.equal(notes, '')
  assert.deepEqual(errors, [
    '#7 has no change type: https://github.com/jack-barr3tt/bouncer/pull/7',
    '#8 has feature and fix: https://github.com/jack-barr3tt/bouncer/pull/8',
  ])
})
