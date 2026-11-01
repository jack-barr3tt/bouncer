import assert from 'node:assert/strict'
import { test } from 'node:test'
import { artifactsForDiff, releaseArtifacts } from './release-publish.mjs'

test('each artifact watches its own inputs', () => {
  assert.deepEqual(artifactsForDiff(['client/src/index.ts']), ['client'])
  assert.deepEqual(artifactsForDiff(['create/src/index.ts']), ['create'])
  assert.deepEqual(artifactsForDiff(['templates/github/pr.yaml']), ['create'])
  assert.deepEqual(artifactsForDiff(['deploy/docker-compose.yml']), ['create', 'image'])
  assert.deepEqual(artifactsForDiff(['deploy/stack.yml']), ['image'])
  assert.deepEqual(artifactsForDiff(['deploy/env.yaml']), ['create', 'image'])
  assert.deepEqual(artifactsForDiff(['scaffold/apps.yaml']), ['create', 'image'])
  assert.deepEqual(artifactsForDiff(['server/cmd/bouncer/main.go']), ['image', 'builder'])
  assert.deepEqual(artifactsForDiff(['hub/src/main.tsx']), ['image'])
  assert.deepEqual(artifactsForDiff(['scripts/docker-entrypoint.sh']), ['image'])
  assert.deepEqual(artifactsForDiff(['Dockerfile']), ['image', 'builder'])
  assert.deepEqual(artifactsForDiff(['.dockerignore']), ['image', 'builder'])
  assert.deepEqual(artifactsForDiff(['scripts/notes.mjs', 'RELEASING.md']), [])
})

test('a mixed diff publishes every affected artifact in a stable order', () => {
  assert.deepEqual(
    artifactsForDiff(['server/go.mod', 'client/package.json', 'hub/package.json', 'create/vendor.ts']),
    releaseArtifacts,
  )
})

test('an empty diff publishes nothing', () => {
  assert.deepEqual(artifactsForDiff([]), [])
})
