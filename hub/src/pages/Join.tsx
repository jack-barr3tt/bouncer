import { useState, type SubmitEvent } from 'react'
import { useRedeemAccessCode } from '../api/generated.ts'
import { visitorId } from '../fingerprint.ts'

export default function Join() {
  const code = decodeURIComponent(window.location.pathname.replace(/^\/code\/?/, ''))
  const [nickname, setNickname] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState(false)
  const redeem = useRedeemAccessCode()

  async function onSubmit(event: SubmitEvent) {
    event.preventDefault()
    setError('')
    setPending(true)
    try {
      const fingerprint = await visitorId()
      await redeem.mutateAsync({ data: { code, nickname, fingerprint } })
      window.location.assign('/')
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Could not join.')
      setPending(false)
    }
  }

  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <main className="mx-auto max-w-md px-4 py-16">
        <p className="text-sm font-medium text-stone-500">Join</p>
        <h1 className="mt-2 text-4xl font-semibold tracking-tight">Choose a nickname</h1>
        <p className="mt-3 text-sm text-stone-600">
          Code <span className="font-mono">{code}</span>
        </p>
        <form className="mt-8 rounded-2xl border border-stone-200 bg-white p-5" onSubmit={(event) => void onSubmit(event)}>
          <label className="block text-sm font-medium" htmlFor="nickname">
            Nickname
          </label>
          <input
            id="nickname"
            className="mt-1 w-full rounded-xl border border-stone-300 px-3 py-2"
            value={nickname}
            onChange={(event) => setNickname(event.target.value)}
            maxLength={40}
            required
          />
          {error ? (
            <p className="mt-4 text-sm text-red-700" role="alert">
              {error}
            </p>
          ) : null}
          <button
            className="mt-5 rounded-xl bg-stone-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
            type="submit"
            disabled={pending}
          >
            {pending ? 'Joining…' : 'Join'}
          </button>
        </form>
      </main>
    </div>
  )
}
