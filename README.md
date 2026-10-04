# Bouncer

Sign-in and deployment for a directory of small static web apps. The homepage, the auth server, and `@jack-barr3tt/bouncer-client` live here. The apps live in a site directory this server mounts.

## Run

From this repository, with a site checkout next to it:

```bash
SITE_ROOT=../apps AUTH_BOOTSTRAP_USERNAME=admin AUTH_BOOTSTRAP_PASSWORD=change-me docker compose up --build
```

The first start writes any missing `AGENTS.md`, `apps.yaml`, and the Hello sample into the site. Hello is already built, so it is on the homepage immediately. Change the bootstrap password before using it anywhere but a local machine.

The site is served at `http://127.0.0.1:8080`. Apps are read from `<site>/apps/<slug>/dist`.

## Client

Apps that need the signed-in name depend on the published client:

```bash
npm install @jack-barr3tt/bouncer-client
```

`scripts/tagbump` cuts the next `vX.Y.Z` tag. Pushing that tag builds `client/src` and publishes `dist/`. Until `0.1.0` is on npm, build it and link it locally: `cd client && npm ci && npm run build && npm link`, then `npm link @jack-barr3tt/bouncer-client` inside the app.

`currentIdentity()` returns `{ kind: 'user', username }` or `{ kind: 'temporary', nickname }`. `watchAccess(slug)` sends the browser home when that app's access is removed.

## Site

In an empty directory:

```bash
npx @jack-barr3tt/create-bouncer
```

Until that package is on npm, run the same command from this repository with `node create/dist/index.js` after `npm ci && npm run build` in `create/`.

It asks which CI to use (GitHub Actions, Woodpecker, or Forgejo) and writes the pipelines, the build and deploy scripts, and `deploy/docker-compose.yml` with `deploy/.env.example`. On the server, copy `deploy/.env.example` to `deploy/.env`, set the bootstrap password, and from `deploy/` run `docker compose up -d`.

To set the same files up by hand, copy `templates/github/`, `templates/woodpecker/`, or `templates/forgejo/` into the site, copy `templates/site/scripts/` to `scripts/`, and copy `deploy/docker-compose.yml` with `deploy/.env.example`. See [AGENTS.md](AGENTS.md) for the secret names.

## Release

A tag `vX.Y.Z` is built by Woodpecker and published as `ghcr.io/jack-barr3tt/bouncer:X.Y.Z`, `@jack-barr3tt/bouncer-client@X.Y.Z`, and `@jack-barr3tt/create-bouncer@X.Y.Z`. See [AGENTS.md](AGENTS.md) for `scripts/tagbump` and the secrets.

`deploy/docker-compose.yml` runs that image with a site mounted. The site repository deploys the apps.
