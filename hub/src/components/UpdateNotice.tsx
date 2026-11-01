import { Alert } from 'flowbite-react'
import { useGetUpdate } from '../api/generated.ts'
import { useSession } from '../hooks/useSession.ts'

export default function UpdateNotice() {
  const { session } = useSession()
  const admin = session?.role === 'admin'
  const query = useGetUpdate({
    query: { enabled: admin, refetchInterval: 60_000 },
  })
  const notice = query.data
  if (!admin || !notice?.pending || !notice.version || !notice.keys?.length) return null
  return (
    <Alert className="mt-6" color="warning">
      Bouncer {notice.version} is available. Set {notice.keys.join(', ')} in deploy/.env. This site stays on the current version until then.
    </Alert>
  )
}
