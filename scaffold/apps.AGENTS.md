# Apps

Each directory here is one app. The platform serves it at `/apps/<slug>/` after `npm run build`.

- Keep the app's dependencies in its own `package.json`.
- Set Vite `base` to `'./'`.
- Register the app in the root `apps.yaml`.
- To read the signed-in name, `npm install @jack-barr3tt/bouncer-client`, create `new Bouncer('<slug>')`, and call `identity()`.
- To leave as soon as access is revoked, call `watch()` on that same object.
- To keep strings for the signed-in person, use `storage.getItem`, `storage.setItem`, `storage.removeItem`, `storage.clear`, `storage.key`, and `storage.length`.

The homepage is not in this directory. Sign-in is Bouncer. Do not add a backend for it.
