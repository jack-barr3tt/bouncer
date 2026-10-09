# @jack-barr3tt/bouncer-client

Identity and live revoke for an app served by [Bouncer](https://github.com/jack-barr3tt/bouncer).

```bash
npm install @jack-barr3tt/bouncer-client
```

```ts
import { currentIdentity, watchAccess } from '@jack-barr3tt/bouncer-client'

const who = await currentIdentity()
// { kind: 'user', username } or { kind: 'temporary', nickname }

watchAccess('your-slug')
```

`currentIdentity()` reads `/api/session` and returns the signed-in username or nickname, or `null` when nobody is signed in.

`watchAccess(slug)` listens on `/api/access/stream`. It sends the browser to the homepage when access to that app is removed, and to the login page when the session ends. Both calls are optional.

While you develop, proxy `/api` to Bouncer. The Vite dev server is not behind the gate.

## License

[MIT](LICENSE)
