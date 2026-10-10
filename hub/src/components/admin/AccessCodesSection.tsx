import { addDays, addHours, format, formatISO, parse } from 'date-fns'
import { useState, useSyncExternalStore, type SubmitEvent } from 'react'
import { useCreateAccessCode, useListAccessCodes, type AppInfo } from '../../api/generated.ts'
import { useAdminAction } from '../../hooks/useAdminAction.ts'
import AccessCodeCard from './AccessCodeCard.tsx'

const localInput = "yyyy-MM-dd'T'HH:mm"

let suggestedExpiry = ''

function suggestedExpirySnapshot() {
  if (suggestedExpiry === '') suggestedExpiry = format(addHours(new Date(), 8), localInput)
  return suggestedExpiry
}

const expiryPresets = [
  { label: '1 hour', at: (now: Date) => addHours(now, 1) },
  { label: '8 hours', at: (now: Date) => addHours(now, 8) },
  { label: '1 day', at: (now: Date) => addDays(now, 1) },
  { label: '7 days', at: (now: Date) => addDays(now, 7) },
]

export default function AccessCodesSection({ apps }: { apps: AppInfo[] }) {
  const { error, run } = useAdminAction()
  const codesQuery = useListAccessCodes()
  const codes = codesQuery.data?.codes ?? []
  const [label, setLabel] = useState('')
  const [maxSignups, setMaxSignups] = useState(1)
  const suggested = useSyncExternalStore(
    () => () => {},
    suggestedExpirySnapshot,
    () => '',
  )
  const [expires, setExpires] = useState<string | null>(null)
  const expiry = expires ?? suggested
  const [codeApps, setCodeApps] = useState<string[]>([])
  const createCode = useCreateAccessCode()
  const loadError = codesQuery.error instanceof Error ? codesQuery.error.message : ''
  const shownError = error || loadError

  async function onCreateCode(event: SubmitEvent) {
    event.preventDefault()
    await run(async () => {
      await createCode.mutateAsync({
        data: {
          label,
          appSlugs: codeApps,
          expiresAt: formatISO(parse(expiry, localInput, new Date())),
          maxSignups,
        },
      })
      setLabel('')
      setCodeApps([])
    })
  }

  function toggle(slug: string) {
    setCodeApps((current) => (current.includes(slug) ? current.filter((item) => item !== slug) : [...current, slug]))
  }

  return (
    <section className="mt-12">
      <h2 className="text-xl font-semibold">Access codes</h2>
      {shownError ? (
        <p className="mt-4 text-sm text-red-700" role="alert">
          {shownError}
        </p>
      ) : null}
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
            value={expiry}
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
          <AccessCodeCard key={code.id} code={code} run={run} />
        ))}
      </ul>
    </section>
  )
}
