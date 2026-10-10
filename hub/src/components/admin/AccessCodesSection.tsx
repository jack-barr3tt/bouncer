import { addDays, addHours, format, formatISO, parse } from 'date-fns'
import { Alert, Button, Card, Checkbox, Label, TextInput } from 'flowbite-react'
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
        <Alert className="mt-4" color="failure">
          {shownError}
        </Alert>
      ) : null}
      <Card className="mt-4">
        <form className="flex flex-col gap-4" onSubmit={(event) => void onCreateCode(event)}>
          <div>
            <Label htmlFor="code-label">Label</Label>
            <TextInput id="code-label" className="mt-1" value={label} onChange={(event) => setLabel(event.target.value)} />
          </div>
          <div className="flex flex-wrap gap-2">
            {expiryPresets.map((preset) => (
              <Button
                key={preset.label}
                color="light"
                pill
                size="xs"
                type="button"
                onClick={() => setExpires(format(preset.at(new Date()), localInput))}
              >
                {preset.label}
              </Button>
            ))}
          </div>
          <div>
            <Label htmlFor="code-expires">Expires</Label>
            <TextInput
              id="code-expires"
              className="mt-1"
              type="datetime-local"
              value={expiry}
              onChange={(event) => setExpires(event.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="code-limit">Signup limit</Label>
            <TextInput
              id="code-limit"
              className="mt-1 max-w-24"
              type="number"
              min={1}
              max={1000}
              value={maxSignups}
              onChange={(event) => setMaxSignups(Number(event.target.value))}
              required
            />
          </div>
          <fieldset>
            <legend className="text-sm font-medium">Apps</legend>
            <div className="mt-2 flex flex-wrap gap-3">
              {apps.map((app) => (
                <Label key={app.slug} className="flex items-center gap-2">
                  <Checkbox checked={codeApps.includes(app.slug)} onChange={() => toggle(app.slug)} />
                  {app.name}
                </Label>
              ))}
            </div>
          </fieldset>
          <Button type="submit">Create code</Button>
        </form>
      </Card>
      <ul className="mt-6 flex flex-col gap-4">
        {codes.map((code) => (
          <AccessCodeCard key={code.id} code={code} run={run} />
        ))}
      </ul>
    </section>
  )
}
