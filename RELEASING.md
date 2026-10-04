# Releasing

Pull requests and pushes to `main` run the server tests, hub lint, the site script tests, and the create-bouncer tests. Pull requests also build the client and Hello.

A pull request has one change-type label: `security`, `feature`, `fix`, `docs`, `test`, `deps`, or `chore`. Create those labels on the repository. The pull request check fails otherwise, and it reruns when the labels change. Other labels can sit alongside that one.

On `main`, with a clean working tree:

```bash
scripts/tagbump patch
```

Use `minor` or `major` instead of `patch` when that is the bump you want. The script lists the merged pull requests since the previous tag and prints release notes. A note is the pull request title linked to that pull request, under the heading for its label. The tag is lightweight. With no tags yet, the count starts at `v0.0.0`, so the first `v0.1.0` is `scripts/tagbump minor`.

Push `main` and the tag. The script prints the exact command. Woodpecker publishes `ghcr.io/jack-barr3tt/bouncer:<version>` and `:latest`, stages `@jack-barr3tt/bouncer-client` and `@jack-barr3tt/create-bouncer`, and opens a GitHub release with those notes.

Approve each staged package on npmjs.com, or with `npm stage approve <id>`. Approval asks for a one-time code. Until you approve, the new versions cannot be installed.

Secrets, set in Woodpecker:

| Secret | Purpose |
| --- | --- |
| `registry_user` | GHCR user |
| `registry_token` | GHCR token |
| `npm_token` | npm token that can stage the client and `create-bouncer` |
| `github_token` | GitHub token that can read pull requests and create releases |
