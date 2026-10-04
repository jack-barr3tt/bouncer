import { useQueries, useQueryClient } from '@tanstack/react-query'
import { addDays, addHours, format, formatISO, parse, parseISO } from 'date-fns'
import { useState, type SubmitEvent } from 'react'
import {
  getListTemporaryAccountsQueryOptions,
  useCloneRepo,
  useCreateAccessCode,
  useCreateApp,
  useCreateUser,
  useGetDeploy,
  useListAccessCodes,
  useListUsers,
  useRevokeAccessCode,
  useRevokeTemporaryAccount,
  useRunDeploy,
  useSetUserApps,
  useUpdateUser,
  type UserRole,
} from './api/generated.ts'
import type { AppEntry } from './registry.ts'

const localInput = "yyyy-MM-dd'T'HH:mm"
const shownTime = 'd MMM yyyy, HH:mm'

const expiryPresets = [
  { label: '1 hour', at: (now: Date) => addHours(now, 1) },
  { label: '8 hours', at: (now: Date) => addHours(now, 8) },
  { label: '1 day', at: (now: Date) => addDays(now, 1) },
  { label: '7 days', at: (now: Date) => addDays(now, 7) },
]

export default function Admin({ apps }: { apps: AppEntry[] }) {
  const queryClient = useQueryClient()
  const usersQuery = useListUsers()
  const codesQuery = useListAccessCodes()
  const deployQuery = useGetDeploy({
    query: {
      refetchInterval: (query) =>
        query.state.data?.repos.some((repo) => repo.latest?.status === 'running') ? 2000 : false,
    },
  })
  const users = usersQuery.data?.users ?? []
  const codes = codesQuery.data?.codes ?? []
  const deploy = deployQuery.data
  const accountQueries = useQueries({
    queries: codes.map((code) => getListTemporaryAccountsQueryOptions(code.id)),
  })
  const accounts = Object.fromEntries(
    codes.map((code, index) => [code.id, accountQueries[index]?.data?.accounts ?? []]),
  )
  const [error, setError] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [label, setLabel] = useState('')
  const [maxSignups, setMaxSignups] = useState(1)
  const [expires, setExpires] = useState(() => format(addHours(new Date(), 8), localInput))
  const [codeApps, setCodeApps] = useState<string[]>([])
  const [cloneRemote, setCloneRemote] = useState('')
  const [cloneName, setCloneName] = useState('')
  const [cloneBranch, setCloneBranch] = useState('main')
  const [detected, setDetected] = useState<{ source: string; slug: string; name: string; description: string; icon: string }[]>([])
  const createUser = useCreateUser()
  const updateUser = useUpdateUser()
  const setUserApps = useSetUserApps()
  const createCode = useCreateAccessCode()
  const revokeCode = useRevokeAccessCode()
  const revokeAccount = useRevokeTemporaryAccount()
  const runDeploy = useRunDeploy()
  const cloneRepo = useCloneRepo()
  const createApp = useCreateApp()
  const loadError = usersQuery.error ?? codesQuery.error ?? deployQuery.error ?? accountQueries.find((query) => query.error)?.error
  const shownError = error || (loadError instanceof Error ? loadError.message : '')

  async function refresh() {
    await queryClient.invalidateQueries({
      predicate: (query) => {
        const key = query.queryKey[0]
        return (
          key === 'apps.yaml' ||
          (typeof key === 'string' &&
            (key.startsWith('/api/users') || key.startsWith('/api/access-codes') || key === '/api/deploy'))
        )
      },
    })
  }

  async function run(action: () => Promise<unknown>) {
    setError('')
    try {
      await action()
      await refresh()
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Request failed.')
    }
  }

  async function onCreateUser(event: SubmitEvent) {
    event.preventDefault()
    await run(async () => {
      await createUser.mutateAsync({ data: { username, password } })
      setUsername('')
      setPassword('')
    })
  }

  async function onCreateCode(event: SubmitEvent) {
    event.preventDefault()
    await run(async () => {
      await createCode.mutateAsync({
        data: {
          label,
          appSlugs: codeApps,
          expiresAt: formatISO(parse(expires, localInput, new Date())),
          maxSignups,
        },
      })
      setLabel('')
      setCodeApps([])
    })
  }

  async function onClone(event: SubmitEvent) {
    event.preventDefault()
    setError('')
    try {
      const result = await cloneRepo.mutateAsync({ data: { remote: cloneRemote, name: cloneName, branch: cloneBranch } })
      setDetected(
        result.apps.map((app) => {
          const slug = app.source.split('/').filter(Boolean).pop() ?? ''
          const name = slug.replace(/(^|-)([a-z])/g, (_, gap: string, letter: string) => `${gap ? ' ' : ''}${letter.toUpperCase()}`)
          return { source: app.source, slug, name, description: '', icon: name.slice(0, 1) || 'A' }
        }),
      )
      setCloneRemote('')
      setCloneName('')
      setCloneBranch('main')
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Request failed.')
    }
  }

  async function onAddApp(index: number) {
    const app = detected[index]
    if (!app) return
    setError('')
    try {
      await createApp.mutateAsync({ data: app })
      setDetected((current) => current.filter((_, item) => item !== index))
      await refresh()
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Request failed.')
    }
  }

  async function onDeploy(remote: string) {
    await run(async () => {
      await runDeploy.mutateAsync({ data: { remote } })
    })
  }

  function updateDetected(index: number, field: 'slug' | 'name' | 'description' | 'icon', value: string) {
    setDetected((current) => current.map((app, item) => (item === index ? { ...app, [field]: value } : app)))
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
        {shownError ? (
          <p className="mt-4 text-sm text-red-700" role="alert">
            {shownError}
          </p>
        ) : null}

        <section className="mt-10">
          <h2 className="text-xl font-semibold">Deploy</h2>
          <form className="mt-4 flex flex-wrap items-end gap-3" onSubmit={(event) => void onClone(event)}>
            <label className="text-sm">
              Git remote
              <input
                className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                value={cloneRemote}
                onChange={(event) => setCloneRemote(event.target.value)}
                placeholder="git@github.com:you/notes.git"
                required
              />
            </label>
            <label className="text-sm">
              Folder
              <input
                className="mt-1 block w-32 rounded-xl border border-stone-300 px-3 py-2"
                value={cloneName}
                onChange={(event) => setCloneName(event.target.value)}
                placeholder="notes"
                required
              />
            </label>
            <label className="text-sm">
              Branch
              <input
                className="mt-1 block w-28 rounded-xl border border-stone-300 px-3 py-2"
                value={cloneBranch}
                onChange={(event) => setCloneBranch(event.target.value)}
                required
              />
            </label>
            <button className="rounded-full bg-stone-900 px-4 py-2 text-sm text-white" type="submit">
              Clone into apps/
            </button>
          </form>
          {detected.length > 0 ? (
            <div className="mt-4 grid gap-4">
              {detected.map((app, index) => (
                <form
                  className="rounded-2xl border border-stone-200 bg-white p-4"
                  key={app.source}
                  onSubmit={(event) => {
                    event.preventDefault()
                    void onAddApp(index)
                  }}
                >
                  <p className="font-mono text-sm">{app.source}</p>
                  <div className="mt-3 flex flex-wrap items-end gap-3">
                    <label className="text-sm">
                      Slug
                      <input
                        className="mt-1 block w-28 rounded-xl border border-stone-300 px-3 py-2"
                        value={app.slug}
                        onChange={(event) => updateDetected(index, 'slug', event.target.value)}
                        required
                      />
                    </label>
                    <label className="text-sm">
                      Name
                      <input
                        className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                        value={app.name}
                        onChange={(event) => updateDetected(index, 'name', event.target.value)}
                        required
                      />
                    </label>
                    <label className="text-sm">
                      Description
                      <input
                        className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                        value={app.description}
                        onChange={(event) => updateDetected(index, 'description', event.target.value)}
                        required
                      />
                    </label>
                    <label className="text-sm">
                      Icon
                      <input
                        className="mt-1 block w-20 rounded-xl border border-stone-300 px-3 py-2"
                        value={app.icon}
                        onChange={(event) => updateDetected(index, 'icon', event.target.value)}
                        required
                      />
                    </label>
                    <button className="rounded-full bg-stone-900 px-4 py-2 text-sm text-white" type="submit">
                      Add app
                    </button>
                  </div>
                </form>
              ))}
            </div>
          ) : null}
          {deploy == null ? null : deploy.enabled ? (
            <div className="mt-4">
              <p className="font-mono text-sm">{deploy.webhookUrl}</p>
              <button className="mt-3 text-sm underline" type="button" onClick={() => void navigator.clipboard.writeText(deploy.webhookUrl)}>
                Copy webhook URL
              </button>
              <div className="mt-4 grid gap-4">
                {deploy.repos.map((repo) => (
                  <div className="rounded-2xl border border-stone-200 bg-white p-4" key={`${repo.path}:${repo.remote}`}>
                    <p className="text-sm">
                      {repo.path ? `${repo.path} · ` : ''}
                      {repo.remote}
                      {repo.branch ? ` · ${repo.branch}` : ''}
                    </p>
                    <div className="mt-3 text-sm">
                      <button
                        className="underline disabled:text-stone-400"
                        type="button"
                        disabled={repo.latest?.status === 'running'}
                        onClick={() => void onDeploy(repo.remote)}
                      >
                        Deploy now
                      </button>
                    </div>
                    {repo.latest ? (
                      <div className="mt-4">
                        <p className="text-sm">
                          {repo.latest.status} · {repo.latest.sha.slice(0, 7)}
                          {repo.latest.finishedAt ? ` · ${format(parseISO(repo.latest.finishedAt), shownTime)}` : ''}
                        </p>
                        {repo.latest.log ? (
                          <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap text-xs text-stone-600">{repo.latest.log}</pre>
                        ) : null}
                      </div>
                    ) : (
                      <p className="mt-4 text-sm text-stone-600">No deploys yet.</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <p className="mt-4 text-sm text-stone-600">Git deploy is off until a clone under apps/ is registered, or GIT_REMOTE is set.</p>
          )}
        </section>

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
                        void run(() => updateUser.mutateAsync({ id: user.id, data: { role: event.target.value as UserRole } }))
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
                    onChange={(event) => void run(() => updateUser.mutateAsync({ id: user.id, data: { disabled: event.target.checked } }))}
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
                              void run(() => setUserApps.mutateAsync({ id: user.id, data: { slugs } }))
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
                      await updateUser.mutateAsync({ id: user.id, data: { password: next } })
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
              {expiryPresets.map((preset) => (
                <button
                  key={preset.label}
                  className="rounded-full border border-stone-300 px-3 py-1 text-sm"
                  type="button"
                  onClick={() => setExpires(format(preset.at(new Date()), localInput))}
                >
                  {preset.label}
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
                  {code.revokedAt ? ' · revoked' : ''} · expires {format(parseISO(code.expiresAt), shownTime)}
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
                    <button className="underline" type="button" onClick={() => void run(() => revokeCode.mutateAsync({ id: code.id }))}>
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
                          onClick={() => void run(() => revokeAccount.mutateAsync({ id: account.id }))}
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
