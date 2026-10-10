import { useSetUserApps, useUpdateUser, type AppInfo, type User, type UserRole } from '../../api/generated.ts'
import type { AdminAction } from '../../hooks/useAdminAction.ts'

export default function UserCard({ user, apps, run }: { user: User; apps: AppInfo[]; run: AdminAction }) {
  const updateUser = useUpdateUser()
  const setUserApps = useSetUserApps()

  return (
    <li className="rounded-2xl border border-stone-200 bg-white p-4">
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
                    const slugs = event.target.checked ? [...user.apps, app.slug] : user.apps.filter((slug) => slug !== app.slug)
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
  )
}
