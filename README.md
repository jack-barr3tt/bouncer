# Bouncer

Have you ever wanted to quickly deploy and share a web app, using your own infrastructure, but didn't want to deal with all that hassle? Bouncer might be just what you're looking for!

Bouncer is a simple way to deploy and share web apps. It runs on your own server and domain, and builds apps with Vite. One deployment is one site directory, with `apps.yaml` as the catalog. Apps can live under `apps/`, or in git clones you put in that directory and point at from `apps.yaml`. Bouncer fetches those clones and publishes the apps that changed. 

Create persistent user accounts, or provide scoped temporary access via a URL, join code or QR code. Stop building auth into all your pet projects - let bouncer do it for you! You can even hook directly into Bouncer's identity system via the `@jack-barr3tt/bouncer-client` library if you want.

## Quick start

Initialize a Bouncer project:

```bash
npx @jack-barr3tt/create-bouncer
```

It asks which CI to use (GitHub Actions, Woodpecker, or Forgejo), the public URL, and the absolute path of this directory on the machine that will run Bouncer. It writes the pipelines, the build and deploy scripts, and `deploy/docker-compose.yml` with `deploy/.env.example`.

Copy `deploy/.env.example` to `deploy/.env`. Set `AUTH_BOOTSTRAP_USERNAME` and `AUTH_BOOTSTRAP_PASSWORD`. From `deploy/`:

```bash
docker compose up -d
```

Open the public URL and sign in with that account. The first start of an empty site writes `apps.yaml` and a Hello sample that is already built, so it is on the homepage immediately. Change the bootstrap password before anyone else can reach the server.

Compose publishes port 8080 on `127.0.0.1`. Put TLS and your public hostname on a proxy in front of that port. The images are `ghcr.io/jack-barr3tt/bouncer` and `ghcr.io/jack-barr3tt/bouncer-builder`. `IMAGE_TAG` selects the tag for both, and defaults to `latest`. `SITE_PATH` is the site directory mounted into the server. The builder is not published on a port.

## Add an app

Add each app as a directory under `apps/` in that same repository. Bouncer serves `apps/<slug>/dist` when that build exists, and `apps/<slug>` otherwise. Register it in `apps.yaml` at the root of the site:

- `slug` — directory name. Lowercase words separated by hyphens.
- `name` — display name, 60 characters or fewer.
- `description` — one sentence, 200 characters or fewer.
- `path` — exactly `/apps/<slug>/`.
- `icon` — a short label or emoji, 32 characters or fewer.

A Vite app should set `base` to `'./'` and build with `npm run build`. Reserved slugs are `hub`, `assets`, and `code`.

A new account can open nothing until an admin turns apps on. Admins can open every app in `apps.yaml`. Delete `apps/hello` and its `apps.yaml` entry when you no longer want the sample.

## Accounts and guests

People sign in on the homepage with a username and password. An admin creates those accounts.

An access code is an invitation. The join URL is `/code/<code>`. Opening it asks for a nickname and creates a temporary account for that browser. The same browser resumes the same temporary account. Revoking the code stops new signups. Revoking the temporary account signs that person out.

## In an app

```bash
npm install @jack-barr3tt/bouncer-client
```

```ts
import { currentIdentity, watchAccess } from '@jack-barr3tt/bouncer-client'

const who = await currentIdentity()
// { kind: 'user', username } or { kind: 'temporary', nickname }

watchAccess('your-slug')
```

`currentIdentity()` returns the signed-in username or nickname, or `null` when nobody is signed in. `watchAccess(slug)` sends the browser home when access to that app is removed, and to the login page when the session ends. Both are optional.

While you develop, proxy `/api` to Bouncer on port 8080. The Vite dev server is not behind the gate.

## Publish from git

Set these in `deploy/.env`, and replace `deploy/git-key` with a read-only deploy key for the site repository:

```
GIT_REMOTE=git@github.com:you/site.git
GIT_BRANCH=main
DEPLOY_WEBHOOK_SECRET=choose-a-long-secret
BUILDER_TOKEN=choose-another-long-secret
```

`BUILDER_TOKEN` is shared with the builder container. It is not a public secret.

In GitHub or Forgejo, add a webhook for push events to `https://apps.example.com/api/hooks/git`, using that webhook secret. Bouncer fetches the commit, builds the apps whose files changed, and publishes `apps.yaml` and each new `dist`. The homepage reads `apps.yaml` on the next request. A failed build leaves the previous files in place.

To deploy only after CI passes, or when the forge cannot call the server, the pipeline posts the commit instead:

```bash
curl -fsS -X POST "$DEPLOY_URL/api/deploy" \
  -H "Authorization: Bearer $DEPLOY_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"sha\":\"$SHA\"}"
```

`DEPLOY_URL` is the public origin, with no path. Woodpecker secrets are `deploy_token` and `deploy_url`. GitHub Actions and Forgejo use `DEPLOY_TOKEN` and `DEPLOY_URL`. Set `DEPLOY_POLL_INTERVAL` (for example `1m`) when nothing can call in. Leave `GIT_REMOTE` empty to keep serving the files already in the site directory.

## Apps from more than one repository

Clone each project into the site directory, then point `apps.yaml` at the Vite app inside it. One clone can hold many apps. One app can be the whole clone. `source` is the site-relative directory that contains `package.json`:

```yaml
apps:
  - slug: notes
    name: Notes
    description: A notebook.
    path: /apps/notes/
    icon: N
    source: apps/notes
  - slug: chess
    name: Chess
    description: A board.
    path: /apps/chess/
    icon: C
    source: apps/games/chess
```

`apps/notes` is a clone of one app. `apps/games` is a clone that contains more than one app. A directory under `apps/` with no git clone stays as it is. Bouncer reads `origin` and the checked-out branch from each clone. A detached HEAD fails that clone's deploy. The deploy key must be able to read every origin.

The admin page can clone a repository into `apps/<folder>` and register the apps it finds, so a new project does not need a shell on the server.

Push webhooks for every forge use the same URL, `https://apps.example.com/api/hooks/git`, and the same `DEPLOY_WEBHOOK_SECRET`. Bouncer matches the repository in the payload to a clone. It publishes `dist` under `apps/<slug>/` and leaves `apps.yaml` as you wrote it. The admin page lists each clone and can publish that branch tip.

When more than one clone is configured, the deploy request includes the remote:

```bash
curl -fsS -X POST "$DEPLOY_URL/api/deploy" \
  -H "Authorization: Bearer $DEPLOY_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"sha\":\"$SHA\",\"remote\":\"git@github.com:you/notes.git\"}"
```

SSH and HTTPS forms of the same repository match. A site that sets `GIT_REMOTE` and does not set `source` still publishes that one repository, including its `apps.yaml`.

The admin page shows the latest deploy and can publish the branch tip. The webhook URL is on that page too. Paste `DEPLOY_WEBHOOK_SECRET` from `deploy/.env` into the forge.

## Set the files up by hand

Copy `templates/github/`, `templates/woodpecker/`, or `templates/forgejo/` into the site, copy `templates/site/scripts/` to `scripts/`, and copy `deploy/docker-compose.yml` with `deploy/.env.example`. Then copy that example to `deploy/.env`, set the public URL, site path, and bootstrap account, and set `COOKIE_SECURE=true` when the public URL is HTTPS. Create `deploy/git-key` before starting Compose so that mount is a file. Run Compose from `deploy/` so that file supplies the image and the site path.
