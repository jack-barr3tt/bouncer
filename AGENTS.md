# Bouncer

Bouncer signs people in and serves a site of static apps. A site is one directory: `apps.yaml` plus the files it serves. One deployment mounts that directory. An app is a folder under `apps/`, or a directory inside a git clone in the site, named by `source` in `apps.yaml`. This repository is the server, the homepage, and the client those apps use. It is not a collection of apps.

How to set up and run a site is in [README.md](README.md). How to cut a release is in [RELEASING.md](RELEASING.md).

## Layout

```
server/          Go server. Module github.com/jack-barr3tt/bouncer
hub/             Homepage, served at /
client/          @jack-barr3tt/bouncer-client. Identity and live revoke
scaffold/        Files written into an empty site on startup
templates/       CI pipelines and site scripts a site copies into place
create/          @jack-barr3tt/create-bouncer. Writes CI files and the Hello sample into a site
server/schema/   JSON Schema for apps.yaml, embedded in the server
compose.yaml     Local Postgres and Bouncer, mounting a site
deploy/          Compose for running the published image with a site mounted
.woodpecker/     Pull request checks, main checks, and release
```

The OpenAPI file is `server/openapi.yaml`. `make generate` rewrites the server from it (`server/oapi-codegen.yaml`) and rewrites the hub client (`hub/orval.config.ts`). Do not edit `server/internal/api/gen.go` or `hub/src/api/generated.ts`.

Schema changes go through numbered files in `server/db/migrations/`. After changing them, run `make schema` to refresh `server/db/schema.sql`. That file is the applied-schema dump. Do not edit it by hand, and do not apply it as a migration.

## Site

`SITE_ROOT` is the consumer directory. `HUB_DIR` is the built homepage. When `HUB_DIR` is unset and `SITE_ROOT/apps/hub` exists, the server uses that build. Otherwise the homepage files are `SITE_ROOT` itself.

App files are `SITE_ROOT/apps/<slug>/dist` when that directory exists, and `SITE_ROOT/apps/<slug>` otherwise. An app with `upstream` is proxied to that origin instead. `/apps.yaml` is always read from `SITE_ROOT`. The server checks that file against the schema in `server/schema/apps.schema.json`.

`ROUTING=subdomain` serves each app at `<slug>.<host>` of `PUBLIC_BASE_URL`. The apex stays the homepage. Unset, `path`, or a localhost or IP public host keeps `/apps/<slug>/`. A wildcard DNS record and certificate have to cover one label under that host.

`create-bouncer` writes `AGENTS.md`, `apps/AGENTS.md`, `apps.yaml`, and the Hello sample in `scaffold/apps/hello/`. An empty site that still has no `apps.yaml` receives those files on startup, including a built `dist`. A site that already has `apps.yaml` is left alone. When Hello is present and `dist` is missing, startup copies the image's build. The schema stays in this repository. Apps depend on `@jack-barr3tt/bouncer-client` from npm.

## Auth

People sign in on the hub with a username and password. An admin can create accounts. A new account can open nothing until an admin turns apps on. Admins can open every app in `apps.yaml`.

An access code is an invitation. The join URL is `/code/<code>`. Opening it asks for a nickname and creates a temporary account tied to that browser fingerprint and IP. The same fingerprint resumes the same temporary account. Revoking the code stops new signups. Revoking a temporary account signs that person out.

The session cookie is `bouncer_session`. Apps learn the signed-in name from `@jack-barr3tt/bouncer-client` (`currentIdentity()`). They leave when access is revoked only if they call `watchAccess(slug)`.

## CI templates

GitHub Actions, Woodpecker, and Forgejo read pipelines from the repository they build. They cannot import a pipeline from this one. The recommended setup is `npx @jack-barr3tt/create-bouncer`. It asks which CI to use and writes the pipelines, `scripts/ci-build.mjs`, `deploy/docker-compose.yml`, and the Hello sample.

To do that by hand, copy `templates/github/`, `templates/woodpecker/`, or `templates/forgejo/` into the site, copy `templates/site/scripts/` to `scripts/`, and copy `deploy/docker-compose.yml` with `deploy/.env.example`. Run Compose from the `deploy/` directory so the env file fills in the image and the site path.

Bouncer fetches the site repository when `GIT_REMOTE` is set and no app sets `source`. It builds the apps that changed and publishes `apps.yaml` and those `dist` directories. When an app sets `source`, that path is a directory inside `apps/`. It is either a git clone of one app (`apps/<name>`) or an app inside a clone of many (`apps/<name>/<app>`). A directory under `apps/` with no clone is left as it is. Bouncer reads a clone's `origin` and checked-out branch, fetches a clean copy, and publishes `dist` into that app directory. Apps that share a clone share one commit. The admin page can clone a repository into `apps/<folder>` and add the apps it finds. The forge webhook is the public origin plus `/api/hooks/git` for every clone. Set `GIT_REMOTE`, a read-only key in `deploy/git-key`, `DEPLOY_WEBHOOK_SECRET`, and `BUILDER_TOKEN`. The same key must be able to read every origin. Pull-request pipelines still build the apps that changed. The deploy pipeline only notifies Bouncer. Woodpecker secrets for that call are `deploy_token` and `deploy_url`. GitHub and Forgejo use `DEPLOY_TOKEN` and `DEPLOY_URL`. When more than one clone is configured, the deploy request includes `remote`. Leave `GIT_REMOTE` empty, and leave `source` unset, to keep serving the files already in the site directory.

## Local run

How to run this repository is in [DEV.md](DEV.md). From the repository root, `make dev` starts Postgres and a tmux session for this checkout. Compose can also start the published server image, which does not include changes in this checkout.

## Checks

`make check` runs the client, create, script, server, hub, and Hello checks. Server tests that use Postgres need `TEST_DATABASE_URL`. They truncate that database.
