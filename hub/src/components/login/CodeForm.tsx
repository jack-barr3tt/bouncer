import { useNavigate } from '@tanstack/react-router'
import { Button, Card, HelperText, Label, TextInput } from 'flowbite-react'
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
    <Card>
      <form className="flex flex-col gap-4" onSubmit={openCode}>
        <div>
          <Label htmlFor="code">Have a code?</Label>
          <HelperText className="mt-1">This opens the join page. It does not sign you in.</HelperText>
          <TextInput
            id="code"
            className="mt-3 font-mono uppercase"
            value={code}
            onChange={(event) => setCode(event.target.value)}
            autoCapitalize="characters"
          />
        </div>
        <Button color="light" type="submit">
          Continue
        </Button>
      </form>
    </Card>
  )
}
