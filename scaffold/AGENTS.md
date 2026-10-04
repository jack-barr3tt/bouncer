# Apps

This directory is a site served by Bouncer. Bouncer serves the homepage, signs people in, and gates each app. You only add apps.

The homepage and the auth server are not in this directory. They run in the Bouncer container. Your apps are the folders under `apps/` in this same git repository, registered in `apps.yaml`. Do not add an app by pointing at another repository.

## Sample

`apps/hello` is included. It reads the signed-in name and leaves when its access is revoked. Delete that directory and its `apps.yaml` entry when you want it gone.

## Add an app

Create `apps/<slug>/` as a Vite + React + TypeScript + Tailwind app. Set Vite `base` to `'./'`. Build with `npm run build` so the files land in `apps/<slug>/dist`. The server reads that `dist` directory.

Add an entry to `apps.yaml`:

- `slug` — directory name. Lowercase words separated by hyphens.
- `name` — display name, 60 characters or fewer.
- `description` — one sentence, 200 characters or fewer.
- `path` — exactly `/apps/<slug>/`.
- `icon` — a short label or emoji, 32 characters or fewer.

Reserved slugs: `hub`, `assets`, `code`. Do not use them. Join links stay at `/code/<code>`.

A new account can open nothing until an admin turns apps on. Admins can open every registered app.

## Identity

Depend on the published client:

```bash
npm install @jack-barr3tt/bouncer-client
```

```ts
import { currentIdentity, watchAccess } from '@jack-barr3tt/bouncer-client'

const who = await currentIdentity()
// { kind: 'user', username } or { kind: 'temporary', nickname }

watchAccess('your-slug')
```

`currentIdentity()` reads `GET /api/session` and returns the signed-in username or nickname. Use that value however the app wants. `watchAccess(slug)` listens for access changes and leaves the page when this app is revoked. Both are optional. Vite dev is not behind the gate; proxy `/api` to the platform (port 8080) so the client can reach it.

Do not copy the client into the app, and do not add another auth server.
