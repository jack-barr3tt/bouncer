import { useNavigate } from '@tanstack/react-router'
import { useState, type SubmitEvent } from 'react'

export default function CodeForm() {
  const navigate = useNavigate()
  const [code, setCode] = useState('')

  function openCode(event: SubmitEvent) {
    event.preventDefault()
    const normalized = code.toUpperCase().replace(/[\s-]/g, '')
    if (!normalized) return
    void navigate({ to: '/code/{-$code}', params: { code: normalized } })
  }

  return (
    <form className="rounded-2xl border border-stone-200 bg-white p-5" onSubmit={openCode}>
      <label className="block text-sm font-medium" htmlFor="code">
        Have a code?
      </label>
      <p className="mt-1 text-sm text-stone-600">This opens the join page. It does not sign you in.</p>
      <input
        id="code"
        className="mt-3 w-full rounded-xl border border-stone-300 px-3 py-2 font-mono uppercase"
        value={code}
        onChange={(event) => setCode(event.target.value)}
        autoCapitalize="characters"
      />
      <button className="mt-4 rounded-xl border border-stone-300 px-4 py-2 text-sm font-medium" type="submit">
        Continue
      </button>
    </form>
  )
}
