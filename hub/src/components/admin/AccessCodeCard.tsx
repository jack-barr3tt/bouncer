import { format, parseISO } from 'date-fns'
import { Alert, Button, Card } from 'flowbite-react'
import { useListTemporaryAccounts, useRevokeAccessCode, useRevokeTemporaryAccount, type AccessCode } from '../../api/generated.ts'
import type { AdminAction } from '../../hooks/useAdminAction.ts'

const shownTime = 'd MMM yyyy, HH:mm'

export default function AccessCodeCard({ code, run }: { code: AccessCode; run: AdminAction }) {
  const accountsQuery = useListTemporaryAccounts(code.id)
  const accounts = accountsQuery.data?.accounts ?? []
  const revokeCode = useRevokeAccessCode()
  const revokeAccount = useRevokeTemporaryAccount()
  const loadError = accountsQuery.error instanceof Error ? accountsQuery.error.message : ''

  return (
    <li>
      <Card>
        <p className="font-medium">{code.label || 'Untitled code'}</p>
        <p className="font-mono text-sm">{code.url}</p>
        <p className="soft text-sm">
          {code.signupCount} / {code.maxSignups} signups
          {code.revokedAt ? ' · revoked' : ''} · expires {format(parseISO(code.expiresAt), shownTime)}
        </p>
        <img className="size-40 rounded-2xl bg-white p-2" src={`/api/access-codes/${code.id}/qr`} alt="" />
        {loadError ? <Alert color="failure">{loadError}</Alert> : null}
        <div className="flex flex-wrap gap-3">
          <Button color="light" size="sm" onClick={() => void navigator.clipboard.writeText(code.url)}>
            Copy link
          </Button>
          {!code.revokedAt ? (
            <Button color="light" size="sm" onClick={() => void run(() => revokeCode.mutateAsync({ id: code.id }))}>
              Revoke code
            </Button>
          ) : null}
        </div>
        <ul className="flex flex-col gap-2">
          {accounts.map((account) => (
            <li key={account.id} className="flex flex-wrap items-center justify-between gap-2 text-sm">
              <span>
                {account.nickname}
                <span className="soft"> · {account.createdIp}</span>
                {account.revokedAt ? ' · revoked' : ''}
              </span>
              {!account.revokedAt ? (
                <Button color="alternative" size="xs" onClick={() => void run(() => revokeAccount.mutateAsync({ id: account.id }))}>
                  Revoke
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      </Card>
    </li>
  )
}
