# Homepage

This is the Bouncer homepage. It is served at `/`. It is not an entry in a site's `apps.yaml`.

The root [AGENTS.md](../AGENTS.md) still applies. Do not turn this app into a shared library, and do not import code from a consumer's `apps/<slug>/`.

## Registry

The Go server reads `apps.yaml`. The hub loads `GET /api/apps`, which returns the slug, name, description, public path, and icon of each app the current session can open. A signed-out request gets an empty list. The yaml file is not served.

The page lists each entry the signed-in account is allowed to open. Apps without a grant are omitted. A temporary account that can open only one app is sent straight to that app. A production build marks an allowed app "Unavailable" when its URL does not respond successfully. `vite dev` does not perform that check.

Sign-in, join (`/code/<code>`), and admin live in this app. `vite dev` proxies `/api` and `/apps` to `http://127.0.0.1:8080`. The hub listens to the access stream and removes a card when that grant disappears.

## Deploy

The homepage is a TanStack Start app. `npm start` serves the production build. The image runs that process and the Go server proxies hub paths to it. `/api` and `/apps` stay on the Go server.

`/assets` belongs to the hub, which is why an app slug cannot be named `assets`.

## Checks

From this directory: `npm run lint` and `npm run build`.
