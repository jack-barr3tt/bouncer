import { useEffect, useState, type FormEvent } from 'react'
import {
  createCode,
  createUser,
  listCodes,
  listTemporaryAccounts,
  listUsers,
  revokeCode,
  revokeTemporaryAccount,
  setUserApps,
  updateUser,
  type AccessCode,
  type Account,
  type AppEntry,
  type TemporaryAccount,
} from './api.ts'

function hoursFromNow(hours: number): string {
  const date = new Date(Date.now() + hours * 60 * 60 * 1000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60 * 1000)
  return local.toISOString().slice(0, 16)
}

export default function Admin({ apps }: { apps: AppEntry[] }) {
  const [users, setUsers] = useState<Account[]>([])
  const [codes, setCodes] = useState<AccessCode[]>([])
  const [accounts, setAccounts] = useState<Record<string, TemporaryAccount[]>>({})
  const [error, setError] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [label, setLabel] = useState('')
  const [maxSignups, setMaxSignups] = useState(1)
  const [expires, setExpires] = useState(hoursFromNow(8))
  const [codeApps, setCodeApps] = useState<string[]>([])

  async function refresh() {
    const [userList, codeList] = await Promise.all([listUsers(), listCodes()])
    setUsers(userList.users)
    setCodes(codeList.codes)
    const grouped = await Promise.all(
      codeList.codes.map(async (code) => [code.id, (await listTemporaryAccounts(code.id)).accounts] as const),
    )
    setAccounts(Object.fromEntries(grouped))
  }

  useEffect(() => {
    let cancelled = false
    refresh().catch((caught: unknown) => {
      if (!cancelled) setError(caught instanceof Error ? caught.message : 'Could not load admin.')
    })
    return () => {
      cancelled = true
    }
  }, [])

  async function run(action: () => Promise<void>) {
    setError('')
    try {
      await action()
      await refresh()
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Request failed.')
    }
  }

  async function onCreateUser(event: FormEvent) {
    event.preventDefault()
    await run(async () => {
      await createUser(username, password)
      setUsername('')
      setPassword('')
    })
  }

  async function onCreateCode(event: FormEvent) {
    event.preventDefault()
    await run(async () => {
      await createCode({
        label,
        appSlugs: codeApps,
        expiresAt: new Date(expires).toISOString(),
        maxSignups,
      })
      setLabel('')
      setCodeApps([])
    })
  }

  function toggle(slug: string) {
    setCodeApps((current) => (current.includes(slug) ? current.filter((item) => item !== slug) : [...current, slug]))
  }

  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <main className="mx-auto max-w-3xl px-4 py-12">
        <a className="text-sm font-medium underline" href="/">
          All apps
        </a>
        <h1 className="mt-3 text-4xl font-semibold tracking-tight">Admin</h1>
        {error ? (
          <p className="mt-4 text-sm text-red-700" role="alert">
            {error}
          </p>
        ) : null}

        <section className="mt-10">
          <h2 className="text-xl font-semibold">Accounts</h2>
          <form className="mt-4 flex flex-wrap items-end gap-3" onSubmit={(event) => void onCreateUser(event)}>
            <label className="text-sm">
              Username
              <input
                className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                required
              />
            </label>
            <label className="text-sm">
              Password
              <input
                className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
              />
            </label>
            <button className="rounded-xl bg-stone-900 px-4 py-2 text-sm font-medium text-white" type="submit">
              Create user
            </button>
          </form>
          <ul className="mt-6 flex flex-col gap-4">
            {users.map((user) => (
              <li key={user.id} className="rounded-2xl border border-stone-200 bg-white p-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <p className="font-medium">{user.username}</p>
                  <label className="text-sm">
                    Role{' '}
                    <select
                      className="rounded-lg border border-stone-300 px-2 py-1"
                      value={user.role}
                      onChange={(event) =>
                        void run(() => updateUser(user.id, { role: event.target.value as 'admin' | 'user' }).then(() => undefined))
                      }
                    >
                      <option value="user">user</option>
                      <option value="admin">admin</option>
                    </select>
                  </label>
                </div>
                <label className="mt-3 flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={user.disabled}
                    onChange={(event) => void run(() => updateUser(user.id, { disabled: event.target.checked }).then(() => undefined))}
                  />
                  Disabled
                </label>
                {user.role === 'admin' ? (
                  <p className="mt-3 text-sm text-stone-600">Admins can open every app.</p>
                ) : (
                  <fieldset className="mt-3">
                    <legend className="text-sm font-medium">Apps</legend>
                    <div className="mt-2 flex flex-wrap gap-3">
                      {apps.map((app) => (
                        <label key={app.slug} className="flex items-center gap-2 text-sm">
                          <input
                            type="checkbox"
                            checked={user.apps.includes(app.slug)}
                            onChange={(event) => {
                              const slugs = event.target.checked
                                ? [...user.apps, app.slug]
                                : user.apps.filter((slug) => slug !== app.slug)
                              void run(() => setUserApps(user.id, slugs).then(() => undefined))
                            }}
                          />
                          {app.name}
                        </label>
                      ))}
                    </div>
                  </fieldset>
                )}
                <form
                  className="mt-3 flex gap-2"
                  onSubmit={(event) => {
                    event.preventDefault()
                    const form = event.currentTarget
                    const input = form.elements.namedItem('password')
                    const next = input instanceof HTMLInputElement ? input.value : ''
                    void run(async () => {
                      await updateUser(user.id, { password: next })
                      form.reset()
                    })
                  }}
                >
                  <input
                    name="password"
                    className="rounded-xl border border-stone-300 px-3 py-2 text-sm"
                    type="password"
                    placeholder="New password"
                    required
                  />
                  <button className="rounded-xl border border-stone-300 px-3 py-2 text-sm" type="submit">
                    Set password
                  </button>
                </form>
              </li>
            ))}
          </ul>
        </section>

        <section className="mt-12">
          <h2 className="text-xl font-semibold">Access codes</h2>
          <form className="mt-4 rounded-2xl border border-stone-200 bg-white p-4" onSubmit={(event) => void onCreateCode(event)}>
            <label className="block text-sm">
              Label
              <input
                className="mt-1 w-full rounded-xl border border-stone-300 px-3 py-2"
                value={label}
                onChange={(event) => setLabel(event.target.value)}
              />
            </label>
            <div className="mt-3 flex flex-wrap gap-2">
              {[
                [1, '1 hour'],
                [8, '8 hours'],
                [24, '1 day'],
                [24 * 7, '7 days'],
              ].map(([hours, name]) => (
                <button
                  key={name}
                  className="rounded-full border border-stone-300 px-3 py-1 text-sm"
                  type="button"
                  onClick={() => setExpires(hoursFromNow(Number(hours)))}
                >
                  {name}
                </button>
              ))}
            </div>
            <label className="mt-3 block text-sm">
              Expires
              <input
                className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                type="datetime-local"
                value={expires}
                onChange={(event) => setExpires(event.target.value)}
                required
              />
            </label>
            <label className="mt-3 block text-sm">
              Signup limit
              <input
                className="mt-1 block w-24 rounded-xl border border-stone-300 px-3 py-2"
                type="number"
                min={1}
                max={1000}
                value={maxSignups}
                onChange={(event) => setMaxSignups(Number(event.target.value))}
                required
              />
            </label>
            <fieldset className="mt-3">
              <legend className="text-sm font-medium">Apps</legend>
              <div className="mt-2 flex flex-wrap gap-3">
                {apps.map((app) => (
                  <label key={app.slug} className="flex items-center gap-2 text-sm">
                    <input type="checkbox" checked={codeApps.includes(app.slug)} onChange={() => toggle(app.slug)} />
                    {app.name}
                  </label>
                ))}
              </div>
            </fieldset>
            <button className="mt-4 rounded-xl bg-stone-900 px-4 py-2 text-sm font-medium text-white" type="submit">
              Create code
            </button>
          </form>
          <ul className="mt-6 flex flex-col gap-4">
            {codes.map((code) => (
              <li key={code.id} className="rounded-2xl border border-stone-200 bg-white p-4">
                <p className="font-medium">{code.label || 'Untitled code'}</p>
                <p className="mt-1 font-mono text-sm">{code.url}</p>
                <p className="mt-1 text-sm text-stone-600">
                  {code.signupCount} / {code.maxSignups} signups
                  {code.revokedAt ? ' · revoked' : ''} · expires {new Date(code.expiresAt).toLocaleString()}
                </p>
                <img className="mt-3 size-40" src={`/api/access-codes/${code.id}/qr`} alt="" />
                <div className="mt-3 flex gap-3 text-sm">
                  <button
                    className="underline"
                    type="button"
                    onClick={() => void navigator.clipboard.writeText(code.url)}
                  >
                    Copy link
                  </button>
                  {!code.revokedAt ? (
                    <button className="underline" type="button" onClick={() => void run(() => revokeCode(code.id).then(() => undefined))}>
                      Revoke code
                    </button>
                  ) : null}
                </div>
                <ul className="mt-4 flex flex-col gap-2">
                  {(accounts[code.id] ?? []).map((account) => (
                    <li key={account.id} className="flex flex-wrap items-center justify-between gap-2 text-sm">
                      <span>
                        {account.nickname}
                        <span className="text-stone-500"> · {account.createdIp}</span>
                        {account.revokedAt ? ' · revoked' : ''}
                      </span>
                      {!account.revokedAt ? (
                        <button
                          className="underline"
                          type="button"
                          onClick={() => void run(() => revokeTemporaryAccount(account.id))}
                        >
                          Revoke
                        </button>
                      ) : null}
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        </section>
      </main>
    </div>
  )
}
