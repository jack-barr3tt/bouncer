# Homepage

This is the Bouncer homepage. It is served at `/`. It is not an entry in a site's `apps.yaml`.

The root [AGENTS.md](../AGENTS.md) still applies. Do not turn this app into a shared library, and do not import code from a consumer's `apps/<slug>/`.

## Registry

`vite.config.ts` reads `SITE_ROOT/apps.yaml` when `SITE_ROOT` is set, then `../apps.yaml`, otherwise `../scaffold/apps.yaml`. Set `APPS_REGISTRY` to point at a different file.

- `vite dev` serves that file at `/apps.yaml`.
- `vite build` copies it into `dist/apps.yaml`.

The running server serves `/apps.yaml` from the mounted site, so a consumer's registry wins over the copy baked into this build.

The page lists each entry the signed-in account is allowed to open. Apps without a grant are omitted. A temporary account that can open only one app is sent straight to that app. A production build marks an allowed app "Unavailable" when its URL does not respond successfully. `vite dev` does not perform that check.

Sign-in, join (`/code/<code>`), and admin live in this app. `vite dev` proxies `/api` and `/apps` to `http://127.0.0.1:8080`. The hub listens to the access stream and removes a card when that grant disappears.

## Deploy

`base` is `'/'`. The image copies `dist/` to `/opt/bouncer/hub`. The `assets/` directory next to `index.html` is why an app slug cannot be named `assets`.

## Checks

From this directory: `npm run lint` and `npm run build`.
