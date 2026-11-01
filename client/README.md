# @jack-barr3tt/bouncer-client

Identity and per-app storage for an app served by [Bouncer](https://github.com/jack-barr3tt/bouncer).

```bash
npm install @jack-barr3tt/bouncer-client
```

```ts
import { Bouncer } from '@jack-barr3tt/bouncer-client'

const bouncer = new Bouncer('your-slug')
const who = await bouncer.identity()
// { kind: 'user', username } or { kind: 'temporary', nickname }

bouncer.watch()

await bouncer.storage.setItem('theme', 'dark')
const theme = await bouncer.storage.getItem('theme')
```

`identity()` reads `/api/session` and returns the signed-in username or nickname, or `null` when nobody is signed in.

`watch()` listens on `/api/access/stream`. It sends the browser to the homepage when access to this app is removed, and to the login page when the session ends.

`storage` is this person's localStorage for the app. `getItem`, `setItem`, `removeItem`, `clear`, `key`, and `length` each return a promise. Values are strings. The library loads and saves the document, and retries when another browser saved in between. `getItem` returns `null` when nobody is signed in. `length()` is a method because it has to read the document. A write that would pass `STORAGE_MAX_BYTES` (1 MiB unless the deployment sets it) rejects with `QuotaExceededError`.

While you develop, proxy `/api` to Bouncer. The Vite dev server is not behind the gate.

## License

[MIT](LICENSE)
