import { useState, type SubmitEvent } from 'react'
import { useCreateUser, useListUsers } from '../../api/generated.ts'
import { useAdminAction } from '../../hooks/useAdminAction.ts'
import UserCard from './UserCard.tsx'

export default function AccountsSection() {
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
        <p className="mt-4 text-sm text-red-700" role="alert">
          {shownError}
        </p>
      ) : null}
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
          <UserCard key={user.id} user={user} run={run} />
        ))}
      </ul>
    </section>
  )
}
