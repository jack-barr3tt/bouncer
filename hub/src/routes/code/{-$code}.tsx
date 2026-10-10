import { useQueryClient } from '@tanstack/react-query'
import { useNavigate, createFileRoute } from '@tanstack/react-router'
import { Alert, Button, Card, Label, TextInput } from 'flowbite-react'
import { useState, type SubmitEvent } from 'react'
import { getGetSessionQueryKey, getListAppsQueryKey, useRedeemAccessCode } from '../../api/generated.ts'
import Fade from '../../components/motion/Fade.tsx'
import Wordmark from '../../components/shell/Wordmark.tsx'
import { visitorId } from '../../fingerprint.ts'

export const Route = createFileRoute('/code/{-$code}')({
  component: Join,
})

function Join() {
  const { code: raw } = Route.useParams()
  const code = raw ? decodeURIComponent(raw) : ''
  const navigate = useNavigate()
  const queryClient = useQueryClient()
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
      const session = await redeem.mutateAsync({ data: { code, nickname, fingerprint } })
      queryClient.setQueryData(getGetSessionQueryKey(), session)
      await queryClient.invalidateQueries({ queryKey: getListAppsQueryKey() })
      await navigate({ to: '/' })
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Could not join.')
      setPending(false)
    }
  }

  return (
    <div>
      <main className="mx-auto max-w-md px-4 py-16">
        <Fade>
          <Wordmark />
          <p className="mt-8 text-xs font-semibold tracking-[0.22em] text-pink-300 uppercase">Join</p>
          <h1 className="mt-2 text-5xl font-extrabold">Choose a nickname</h1>
          <p className="mt-3 text-sm text-pink-100/70">
            Code <span className="font-mono text-pink-200">{code}</span>
          </p>
        </Fade>
        <Fade delay={0.08}>
          <Card className="mt-8">
            <form className="flex flex-col gap-4" onSubmit={(event) => void onSubmit(event)}>
              <div>
                <Label htmlFor="nickname">Nickname</Label>
                <TextInput
                  id="nickname"
                  className="mt-1"
                  value={nickname}
                  onChange={(event) => setNickname(event.target.value)}
                  maxLength={40}
                  required
                />
              </div>
              {error ? <Alert color="failure">{error}</Alert> : null}
              <Button type="submit" disabled={pending}>
                {pending ? 'Joining…' : 'Join'}
              </Button>
            </form>
          </Card>
        </Fade>
      </main>
    </div>
  )
}
