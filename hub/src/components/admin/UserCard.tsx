import { Button, Card, Checkbox, Label, Select, Table, TableBody, TableCell, TableHead, TableHeadCell, TableRow, TextInput } from 'flowbite-react'
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
          <p className="soft text-sm">Admins can open every app.</p>
        ) : (
          <Table hoverable>
            <TableHead>
              <TableRow>
                <TableHeadCell>App</TableHeadCell>
                <TableHeadCell className="w-44">Access</TableHeadCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {apps.map((app) => {
                const mode = user.shareApps.includes(app.slug) ? 'share' : user.apps.includes(app.slug) ? 'view' : 'none'
                return (
                  <TableRow key={app.slug}>
                    <TableCell>
                      <Label htmlFor={`app-${user.id}-${app.slug}`}>{app.name}</Label>
                    </TableCell>
                    <TableCell>
                      <Select
                        id={`app-${user.id}-${app.slug}`}
                        sizing="sm"
                        value={mode}
                        onChange={(event) => {
                          const next = event.target.value
                          const open = user.apps.filter((slug) => slug !== app.slug)
                          const share = user.shareApps.filter((slug) => slug !== app.slug)
                          if (next === 'view' || next === 'share') open.push(app.slug)
                          if (next === 'share') share.push(app.slug)
                          void run(() =>
                            setUserApps.mutateAsync({
                              id: user.id,
                              data: { grants: open.map((slug) => ({ slug, canShare: share.includes(slug) })) },
                            }),
                          )
                        }}
                      >
                        <option value="none">No access</option>
                        <option value="view">View</option>
                        <option value="share">View and share</option>
                      </Select>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
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
          <TextInput name="password" type="password" placeholder="New password" required />
          <Button color="light" type="submit">
            Set password
          </Button>
        </form>
      </Card>
    </li>
  )
}
