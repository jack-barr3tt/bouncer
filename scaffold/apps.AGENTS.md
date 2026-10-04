# Apps

Each directory here is one app. The platform serves it at `/apps/<slug>/` after `npm run build`.

- Keep the app's dependencies in its own `package.json`.
- Set Vite `base` to `'./'`.
- Register the app in the root `apps.yaml`.
- To read the signed-in name, `npm install @jack-barr3tt/bouncer-client` and call `currentIdentity()`.
- To leave as soon as access is revoked, call `watchAccess('<slug>')` from that same package.

The homepage is not in this directory. Sign-in is Bouncer. Do not add a backend for it.
