# Homepage

This is the Bouncer homepage. It is served at `/`. It is not an entry in a site's `apps.yaml`.

The root [AGENTS.md](../AGENTS.md) still applies. Do not turn this app into a shared library, and do not import code from a consumer's `apps/<slug>/`.

## Registry

The Go server reads `apps.yaml`. The hub loads `GET /api/apps`, which returns the slug, name, description, public path, icon, and whether the app is public. A signed-out request returns public apps. The yaml file is not served.

The page lists each app the current visitor can open. A public app is listed for everyone. Other apps are omitted until an account has a grant. A temporary account that can open only one private app is sent straight to that app. Public apps do not count toward that. A production build marks an allowed app "Unavailable" when its URL does not respond successfully. `vite dev` does not perform that check.

Sign-in, join (`/code/<code>`), and admin live in this app. An account allowed to share access codes creates and revokes its own codes from the homepage. `vite dev` proxies `/api` and `/apps` to `http://127.0.0.1:8080`. The hub listens to the access stream and removes a card when that grant disappears.

## Deploy

The homepage is a TanStack Start app. The browser loads the session and `GET /api/apps` through the generated client. `npm start` serves the production build. The image runs that process and the Go server proxies hub paths to it. `/api` and `/apps` stay on the Go server.

`/assets` belongs to the hub, which is why an app slug cannot be named `assets`.

## Checks

From this directory: `npm run lint` and `npm run build`.
