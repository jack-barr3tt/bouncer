import { Button, Card, Checkbox, Label, Select, TextInput } from 'flowbite-react'
import { useSetUserApps, useUpdateUser, type AppInfo, type User, type UserRole } from '../../api/generated.ts'
import type { AdminAction } from '../../hooks/useAdminAction.ts'

export default function UserCard({ user, apps, run }: { user: User; apps: AppInfo[]; run: AdminAction }) {
  const updateUser = useUpdateUser()
  const setUserApps = useSetUserApps()

  return (
    <li>
      <Card>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="font-medium">{user.username}</p>
          <div className="flex items-center gap-2">
            <Label htmlFor={`role-${user.id}`}>Role</Label>
            <Select
              id={`role-${user.id}`}
              sizing="sm"
              value={user.role}
              onChange={(event) =>
                void run(() => updateUser.mutateAsync({ id: user.id, data: { role: event.target.value as UserRole } }))
              }
            >
              <option value="user">user</option>
              <option value="admin">admin</option>
            </Select>
          </div>
        </div>
        <Label className="flex items-center gap-2">
          <Checkbox
            checked={user.disabled}
            onChange={(event) => void run(() => updateUser.mutateAsync({ id: user.id, data: { disabled: event.target.checked } }))}
          />
          Disabled
        </Label>
        {user.role === 'admin' ? (
          <p className="text-sm text-pink-100/70">Admins can open every app.</p>
        ) : (
          <fieldset>
            <legend className="text-sm font-medium">Apps</legend>
            <div className="mt-2 flex flex-wrap gap-3">
              {apps.map((app) => (
                <Label key={app.slug} className="flex items-center gap-2">
                  <Checkbox
                    checked={user.apps.includes(app.slug)}
                    onChange={(event) => {
                      const slugs = event.target.checked ? [...user.apps, app.slug] : user.apps.filter((slug) => slug !== app.slug)
                      void run(() => setUserApps.mutateAsync({ id: user.id, data: { slugs } }))
                    }}
                  />
                  {app.name}
                </Label>
              ))}
            </div>
          </fieldset>
        )}
        <form
          className="flex flex-wrap items-end gap-2"
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
          <TextInput name="password" type="password" placeholder="New password" required sizing="sm" />
          <Button color="light" type="submit">
            Set password
          </Button>
        </form>
      </Card>
    </li>
  )
}
