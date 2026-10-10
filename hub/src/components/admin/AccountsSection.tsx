import { Alert, Button, Label, TextInput } from 'flowbite-react'
import { useState, type SubmitEvent } from 'react'
import { useCreateUser, useListUsers, type AppInfo } from '../../api/generated.ts'
import { useAdminAction } from '../../hooks/useAdminAction.ts'
import UserCard from './UserCard.tsx'

export default function AccountsSection({ apps }: { apps: AppInfo[] }) {
  const { error, run } = useAdminAction()
  const usersQuery = useListUsers()
  const users = usersQuery.data?.users ?? []
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const createUser = useCreateUser()
  const loadError = usersQuery.error instanceof Error ? usersQuery.error.message : ''
  const shownError = error || loadError

  async function onCreateUser(event: SubmitEvent) {
    event.preventDefault()
    await run(async () => {
      await createUser.mutateAsync({ data: { username, password } })
      setUsername('')
      setPassword('')
    })
  }

  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Accounts</h2>
      {shownError ? (
        <Alert className="mt-4" color="failure">
          {shownError}
        </Alert>
      ) : null}
      <form className="mt-4 flex flex-wrap items-end gap-3" onSubmit={(event) => void onCreateUser(event)}>
        <div>
          <Label htmlFor="new-username">Username</Label>
          <TextInput
            id="new-username"
            className="mt-1"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            required
          />
        </div>
        <div>
          <Label htmlFor="new-password">Password</Label>
          <TextInput
            id="new-password"
            className="mt-1"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </div>
        <Button type="submit">Create user</Button>
      </form>
      <ul className="mt-6 flex flex-col gap-4">
        {users.map((user) => (
          <UserCard key={user.id} user={user} apps={apps} run={run} />
        ))}
      </ul>
    </section>
  )
}
