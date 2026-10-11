# Bouncer

Have you ever wanted to quickly deploy and share a web app, using your own infrastructure, but didn't want to deal with all that hassle? Bouncer might be just what you're looking for!

Bouncer is a simple way to deploy and share web apps. It runs on your own server and domain, and builds apps with Vite. One deployment is one site directory, with `apps.yaml` as the catalog. Apps can live under `apps/`, or be referenced from `apps.yaml` by an external repository's commit or branch. Bouncer fetches that revision and publishes the apps that changed. 

Create persistent user accounts, or provide scoped temporary access via a URL, join code or QR code. Stop building auth into all your pet projects - let bouncer do it for you! You can even hook directly into Bouncer's identity system via the `@jack-barr3tt/bouncer-client` library if you want.

## Quick start

You should make a git repository for your apps, and initialise it with bouncer:

```bash
npx @jack-barr3tt/create-bouncer
```

`.` writes into the repository you are in. The command asks which CI to use (GitHub Actions, Woodpecker, or Forgejo), the public URL, the absolute path where this repository will be checked out on the server, and the host port. That path is saved as `SITE_PATH`. The port is saved as `PORT` and defaults to 8080. It writes the pipelines, the build and deploy scripts, `deploy/`, and the Hello sample at `apps/hello` with its `apps.yaml` entry.

The server gets a clone of this repository. `deploy/` stays inside it and holds `docker-compose.yml`, `.env.example`, and an empty `git-key`. Clone the repository on the server at the path you entered. From `deploy/`:

```bash
cp .env.example .env
# set AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD
docker compose up -d
```

Open the public URL and sign in with that account. The first start copies a built Hello into `apps/hello/dist` when that build is missing, so the sample is on the homepage immediately. Change the bootstrap password before anyone else can reach the server.

Compose publishes `PORT` from `deploy/.env` on `127.0.0.1`. The default is 8080. Change `PORT` when that port is taken, and point your proxy at the port you chose. The images are `ghcr.io/jack-barr3tt/bouncer` and `ghcr.io/jack-barr3tt/bouncer-builder`. `IMAGE_TAG` selects the tag for both, and defaults to `latest`. `SITE_PATH` is the site directory mounted into the server. The builder is not published on a port.

## Add an app

Add each app as a directory under `apps/` in that same repository. Bouncer serves `apps/<slug>/dist` when that build exists, and `apps/<slug>` otherwise. Register it in `apps.yaml` at the root of the site:

- `slug` — directory name. Lowercase words separated by hyphens.
- `name` — display name, 60 characters or fewer.
- `description` — one sentence, 200 characters or fewer.
- `path` — exactly `/apps/<slug>/`.
- `icon` — a short label or emoji, 32 characters or fewer.
- `upstream` — origin to proxy, such as `http://127.0.0.1:3000`. The process must already be listening. Omit it to serve files. It cannot be combined with `source`.

A Vite app should set `base` to `'./'` and build with `npm run build`. Reserved slugs are `hub`, `assets`, and `code`.

Set `ROUTING=subdomain` to serve an app at `https://<slug>.<host>/` instead of `/apps/<slug>/`, using the host of `PUBLIC_BASE_URL`. The homepage stays on that host. `/apps/<slug>/` redirects there. A wildcard DNS record and a wildcard certificate must cover one label under that host: `*.example.com` does not cover `hello.apps.example.com`. Localhost and IP addresses stay on path routing. A proxied app does not own `/api`; Bouncer serves that on the app host.

A new account can open nothing until an admin turns apps on. Admins can open every app in `apps.yaml`. For each other account, an admin chooses per app whether it can only open that app, or can also share temporary access codes for it. Delete `apps/hello` and its `apps.yaml` entry when you no longer want the sample.

## Accounts and guests

People sign in on the homepage with a username and password. An admin creates those accounts.

An access code is an invitation owned by the account that created it. The join URL is `/code/<code>`. Opening it asks for a nickname and creates a temporary account for that browser. The same browser resumes the same temporary account. When that account can open only one app, the homepage sends them straight to it. Revoking the code stops new signups. An admin can revoke any code. The owner can revoke only their own. Revoking the temporary account signs that person out.

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

`currentIdentity()` returns the signed-in username or nickname, or `null` when nobody is signed in. `watchAccess(slug)` sends the browser to the homepage when access to that app is removed, and to the login page when the session ends. Both are optional.

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

In GitHub or Forgejo, add a webhook for push events to `https://apps.example.com/api/hooks/git`, using that webhook secret. Bouncer fetches the commit, builds the apps whose files changed, and publishes `apps.yaml` and each new `dist`. The homepage reads `GET /api/apps` on the next request. A failed build leaves the previous files in place.

To deploy only after CI passes, or when the forge cannot call the server, the pipeline posts the commit instead:

```bash
curl -fsS -X POST "$DEPLOY_URL/api/deploy" \
  -H "Authorization: Bearer $DEPLOY_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"sha\":\"$SHA\"}"
```

`DEPLOY_URL` is the public origin, with no path. Woodpecker secrets are `deploy_token` and `deploy_url`. GitHub Actions and Forgejo use `DEPLOY_TOKEN` and `DEPLOY_URL`. Set `DEPLOY_POLL_INTERVAL` (for example `1m`) when nothing can call in. Leave `GIT_REMOTE` empty to keep serving the files already in the site directory.

## Apps from more than one repository

Apps you write stay in this repository, under `apps/`. `source` is the directory of a Vite app in this repository when it is not `apps/<slug>`. An external repository is a reference in `apps.yaml`, not a clone in the site tree. `commit` pins an exact revision. `branch` tracks that branch tip. An app sets `repo` to the reference name, and `dir` when `package.json` is not at the repository root. Several apps can share one reference and one revision.

```yaml
repos:
  - name: games
    remote: git@github.com:you/games.git
    commit: abcdef0123456789abcdef0123456789abcdef01
  - name: notes
    remote: git@github.com:you/notes.git
    branch: main
apps:
  - slug: chess
    name: Chess
    description: A board.
    path: /apps/chess/
    icon: C
    repo: games
    dir: chess
  - slug: notes
    name: Notes
    description: A notebook.
    path: /apps/notes/
    icon: N
    repo: notes
```

Bouncer checks that revision out in its build workspace and publishes `apps/<slug>/dist`. A new pinned version goes live when `commit` changes. A tracked branch goes live from a push webhook, a poll, or a deploy request. The deploy key must be able to read every origin. `apps.yaml` is left as you wrote it.

The admin page can look up a remote and add the apps it finds. From a shell, the same write is:

```bash
bouncer repo add git@github.com:you/notes.git --branch main
bouncer repo add git@github.com:you/games.git --commit abcdef0123456789abcdef0123456789abcdef01 --name games --app chess=chess
```

On a terminal it asks for anything you leave out. With every flag set, it does not ask. On the server, from `deploy/`:

```bash
docker compose exec bouncer bouncer repo add git@github.com:you/notes.git --branch main
```

Against a checkout of the site, pass `--site` or set `SITE_ROOT` to that directory. If that directory is a git checkout, commit `apps.yaml`. When `GIT_REMOTE` is set, the next site publish replaces `apps.yaml` from the site repository, so the committed file is the one that sticks. The admin page writes the live site directory, which is the right place when the site is not published from git.

Push webhooks for every forge use the same URL, `https://apps.example.com/api/hooks/git`, and the same `DEPLOY_WEBHOOK_SECRET`. Bouncer matches the repository in the payload to a tracked branch. A pinned repository ignores those pushes. The admin page lists each repository and can publish it.

When more than one remote is configured, the deploy request includes the remote:

```bash
curl -fsS -X POST "$DEPLOY_URL/api/deploy" \
  -H "Authorization: Bearer $DEPLOY_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"sha\":\"$SHA\",\"remote\":\"git@github.com:you/notes.git\"}"
```

SSH and HTTPS forms of the same repository match. A site that sets `GIT_REMOTE` still publishes that repository, including its `apps.yaml`, and publishes any pinned repository whose commit changed.

The admin page shows the latest deploy and can publish the branch tip. The webhook URL is on that page too. Paste `DEPLOY_WEBHOOK_SECRET` from `deploy/.env` into the forge.

## Set the files up by hand

Copy `templates/github/`, `templates/woodpecker/`, or `templates/forgejo/` into the site, copy `templates/site/scripts/` to `scripts/`, and copy `deploy/docker-compose.yml` with `deploy/.env.example`. Then copy that example to `deploy/.env`, set the public URL, site path, host port, and bootstrap account, and set `COOKIE_SECURE=true` when the public URL is HTTPS. Create `deploy/git-key` before starting Compose so that mount is a file. Run Compose from `deploy/` so that file supplies the image and the site path.
