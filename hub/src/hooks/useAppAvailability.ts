import { useEffect, useState } from 'react'
import type { AppInfo, Session } from '../api/generated.ts'

type Availability = 'up' | 'down'

function requestSignal(parent: AbortSignal): AbortSignal {
  if (typeof AbortSignal.any === 'function' && typeof AbortSignal.timeout === 'function') {
    return AbortSignal.any([parent, AbortSignal.timeout(5000)])
  }
  return parent
}

async function checkApp(path: string, parent: AbortSignal): Promise<boolean | null> {
  let url: URL
  try {
    url = new URL(path, window.location.href)
  } catch {
    return false
  }
  const sameOrigin = url.origin === window.location.origin
  try {
    const response = await fetch(url, { signal: requestSignal(parent), cache: 'no-store' })
    if (!sameOrigin && response.type === 'opaque') return null
    return response.ok
  } catch {
    if (parent.aborted) return null
    return sameOrigin ? false : null
  }
}

export function useAppAvailability(session: Session | null, apps: AppInfo[]) {
  const [availability, setAvailability] = useState<Record<string, Availability>>({})

  useEffect(() => {
    if (!session || !import.meta.env.PROD) return
    const controller = new AbortController()
    for (const app of apps) {
      void checkApp(app.path, controller.signal).then((up) => {
        if (controller.signal.aborted || up === null) return
        setAvailability((current) => ({ ...current, [app.slug]: up ? 'up' : 'down' }))
      })
    }
    return () => controller.abort()
  }, [session, apps])

  return availability
}
