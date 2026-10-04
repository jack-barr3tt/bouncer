# Bouncer

Bouncer signs people in and serves a site of static apps. A site is a directory of apps plus `apps.yaml`. This repository is the server, the homepage, and the client those apps use. It is not a collection of apps.

## Layout

```
server/          Go server. Module github.com/jack-barr3tt/bouncer
hub/             Homepage, served at /
client/          @jack-barr3tt/bouncer-client. Identity and live revoke
scaffold/        Files written into an empty site on startup
templates/       CI pipelines and site scripts a site copies into place
create/          @jack-barr3tt/create-bouncer. Writes those files into a site
server/schema/   JSON Schema for apps.yaml, embedded in the server
compose.yaml     Local Postgres and Bouncer, mounting a site
deploy/          Compose for running the published image with a site mounted
.woodpecker/     Pull request checks, main checks, and release
```

The OpenAPI file is `server/openapi.yaml`. Regenerate with `oapi-codegen` (`server/oapi-codegen.yaml`). Do not edit `server/internal/api/gen.go`.

## Site

`SITE_ROOT` is the consumer directory. `HUB_DIR` is the built homepage. When `HUB_DIR` is unset and `SITE_ROOT/apps/hub` exists, the server uses that build. Otherwise the homepage files are `SITE_ROOT` itself.

App files are `SITE_ROOT/apps/<slug>/dist` when that directory exists, and `SITE_ROOT/apps/<slug>` otherwise. `/apps.yaml` is always read from `SITE_ROOT`. The server checks that file against the schema in `server/schema/apps.schema.json`.

An empty site receives `AGENTS.md`, `apps/AGENTS.md`, `apps.yaml`, and the Hello sample in `scaffold/apps/hello/` on startup, including a built `dist`. A site that already has `apps.yaml` is left alone. The schema stays in this repository. Apps depend on `@jack-barr3tt/bouncer-client` from npm.

## Auth

People sign in on the hub with a username and password. An admin can create accounts. A new account can open nothing until an admin turns apps on. Admins can open every app in `apps.yaml`.

An access code is an invitation. The join URL is `/code/<code>`. Opening it asks for a nickname and creates a temporary account tied to that browser fingerprint and IP. The same fingerprint resumes the same temporary account. Revoking the code stops new signups. Revoking a temporary account signs that person out.

The session cookie is `bouncer_session`. Apps learn the signed-in name from `@jack-barr3tt/bouncer-client` (`currentIdentity()`). They leave when access is revoked only if they call `watchAccess(slug)`.

## CI templates

GitHub Actions, Woodpecker, and Forgejo read pipelines from the repository they build. They cannot import a pipeline from this one. The recommended setup is `npx @jack-barr3tt/create-bouncer`. It asks which CI to use and writes the pipelines, `scripts/ci-build.mjs`, `scripts/ci-deploy.sh`, and `deploy/docker-compose.yml`.

To do that by hand, copy `templates/github/`, `templates/woodpecker/`, or `templates/forgejo/` into the site, copy `templates/site/scripts/` to `scripts/`, and copy `deploy/docker-compose.yml` with `deploy/.env.example`. Run Compose from the `deploy/` directory so the env file fills in the image and the site path.

Each pipeline builds the apps that changed and publishes their `dist` directories with `scripts/ci-deploy.sh`. The site installs `@jack-barr3tt/bouncer-client` from npm. Woodpecker secrets are `deploy_ssh_key`, `deploy_host`, `deploy_user`, and `deploy_path`. The Woodpecker key is the raw private key. GitHub and Forgejo use `DEPLOY_SSH_KEY`, `DEPLOY_HOST`, `DEPLOY_USER`, and `DEPLOY_PATH`. Those keys are base64-encoded.

## Local run

```bash
SITE_ROOT=../apps AUTH_BOOTSTRAP_USERNAME=admin AUTH_BOOTSTRAP_PASSWORD=change-me docker compose up
```

`SITE_ROOT` defaults to `../apps`. Postgres is published on port 5436.

## Release

Pull requests and pushes to `main` run the server tests, hub lint, the site script tests, and the create-bouncer tests. Pull requests also build the client and Hello. A pull request has one change-type label: `security`, `feature`, `fix`, `docs`, `test`, `deps`, or `chore`. Create those labels on the repository. The pull request check fails otherwise, and it reruns when the labels change. Other labels can sit alongside that one.

`scripts/tagbump patch` (or `minor`, or `major`) lists the merged pull requests since the previous tag and prints release notes. A note is the pull request title linked to that pull request, under the heading for its label. The tag is lightweight. With no tags yet, the count starts at `v0.0.0`, so the first `v0.1.0` is `scripts/tagbump minor`.

Push the tag. Woodpecker publishes `ghcr.io/jack-barr3tt/bouncer:<version>`, stages `@jack-barr3tt/bouncer-client` and `@jack-barr3tt/create-bouncer`, and opens a GitHub release with those notes. Approve each staged package on npmjs.com, or with `npm stage approve <id>`. Approval asks for a one-time code. The site repository deploys its apps.

Secrets, set in Woodpecker:

| Secret | Purpose |
| --- | --- |
| `registry_user` | GHCR user |
| `registry_token` | GHCR token |
| `npm_token` | npm token that can stage the client and `create-bouncer` |
| `github_token` | GitHub token that can read pull requests and create releases |

`deploy/docker-compose.yml` runs the published image. `deploy/.env` comes from `deploy/.env.example`. `SITE_PATH` in that file is the consumer site directory mounted into the container.

## Checks

```bash
cd client && npm ci && npm run build
cd create && npm ci && npm test
node --test scripts/*.test.mjs templates/site/scripts/*.test.mjs
cd server && go test ./...
cd hub && npm ci && npm run lint && npm run build
cd scaffold/apps/hello && npm ci && npm run lint && npm run build
```
