import { useEffect, useState } from 'react'
import { currentIdentity, watchAccess, type Identity } from '@jack-barr3tt/bouncer-client'

export default function App() {
  const [who, setWho] = useState<Identity | null>(null)

  useEffect(() => {
    watchAccess('hello')
    void currentIdentity().then(setWho)
  }, [])

  const name = who?.kind === 'user' ? who.username : who?.nickname

  return (
    <main className="flex min-h-svh items-center justify-center bg-white px-6 text-stone-900">
      <div className="max-w-md text-center">
        <p className="text-sm font-medium text-stone-500">Sample app</p>
        <h1 className="mt-2 text-4xl font-semibold tracking-tight">Hello</h1>
        <p className="mt-4 text-stone-600">
          This app ships with Bouncer. It has its own folder and its own dependencies, and it knows who is signed in.
        </p>
        <p className="mt-4 text-sm text-stone-500">{who && name ? `${who.kind} · ${name}` : 'Not signed in'}</p>
        <a
          className="mt-8 inline-block text-sm font-medium text-stone-900 underline decoration-stone-300 underline-offset-4"
          href="/"
        >
          All apps
        </a>
      </div>
    </main>
  )
}
