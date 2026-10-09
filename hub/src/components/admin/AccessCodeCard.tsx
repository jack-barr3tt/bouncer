import { format, parseISO } from 'date-fns'
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
    <li className="rounded-2xl border border-stone-200 bg-white p-4">
      <p className="font-medium">{code.label || 'Untitled code'}</p>
      <p className="mt-1 font-mono text-sm">{code.url}</p>
      <p className="mt-1 text-sm text-stone-600">
        {code.signupCount} / {code.maxSignups} signups
        {code.revokedAt ? ' · revoked' : ''} · expires {format(parseISO(code.expiresAt), shownTime)}
      </p>
      <img className="mt-3 size-40" src={`/api/access-codes/${code.id}/qr`} alt="" />
      {loadError ? (
        <p className="mt-3 text-sm text-red-700" role="alert">
          {loadError}
        </p>
      ) : null}
      <div className="mt-3 flex gap-3 text-sm">
        <button className="underline" type="button" onClick={() => void navigator.clipboard.writeText(code.url)}>
          Copy link
        </button>
        {!code.revokedAt ? (
          <button className="underline" type="button" onClick={() => void run(() => revokeCode.mutateAsync({ id: code.id }))}>
            Revoke code
          </button>
        ) : null}
      </div>
      <ul className="mt-4 flex flex-col gap-2">
        {accounts.map((account) => (
          <li key={account.id} className="flex flex-wrap items-center justify-between gap-2 text-sm">
            <span>
              {account.nickname}
              <span className="text-stone-500"> · {account.createdIp}</span>
              {account.revokedAt ? ' · revoked' : ''}
            </span>
            {!account.revokedAt ? (
              <button className="underline" type="button" onClick={() => void run(() => revokeAccount.mutateAsync({ id: account.id }))}>
                Revoke
              </button>
            ) : null}
          </li>
        ))}
      </ul>
    </li>
  )
}
