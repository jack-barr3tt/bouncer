# Releasing

Pull requests and pushes to `main` lint the server and the JavaScript, run the server tests, lint the hub, and run the site script tests and the create-bouncer tests. Pull requests also build the client and Hello.

A pull request has one change-type label: `security`, `feature`, `fix`, `docs`, `test`, `deps`, or `chore`. Create those labels on the repository. The pull request check fails otherwise, and it reruns when the labels change. Other labels can sit alongside that one.

On `main`, with a clean working tree:

```bash
scripts/tagbump
```

The script lists the merged pull requests since the previous tag, then asks for `patch`, `minor`, or `major`. A note is the pull request title linked to that pull request, under the heading for its label. The tag is lightweight. With no tags yet, the count starts at `v0.0.0`.

Push `main` and the tag. The script prints the exact command. Woodpecker opens a GitHub release with those notes, then publishes only the artifacts that changed since the previous tag. The first tag publishes all of them. Each published artifact uses that tag's version, so a skipped one has a gap.

`@jack-barr3tt/bouncer-client` is staged when `client/` changed. `@jack-barr3tt/create-bouncer` is staged when `create/`, `templates/`, `deploy/docker-compose.yml`, or `scaffold/` changed. `ghcr.io/jack-barr3tt/bouncer:<version>` and `:latest` are published when the server, the hub, `scaffold/`, `scripts/docker-entrypoint.sh`, the Dockerfile, or `.dockerignore` changed. `ghcr.io/jack-barr3tt/bouncer-builder` gets the same tags when the server, the Dockerfile, or `.dockerignore` changed. `:latest` stays on the last image that was published.

Approve each staged package on npmjs.com, or with `npm stage approve <id>`. Approval asks for a one-time code. Until you approve, the new versions cannot be installed.

Secrets, set in Woodpecker:

| Secret | Purpose |
| --- | --- |
| `registry_user` | GHCR user |
| `registry_token` | GHCR token |
| `npm_token` | npm token that can stage the client and `create-bouncer` |
| `github_token` | GitHub token that can read pull requests and create releases |
