import { useEffect, useRef, useSyncExternalStore } from 'react'
import { toast, Toaster } from 'sonner'
import { subscribeTheme, themeIsDark } from './themeMode.ts'

export function reportError(caught: unknown, fallback = 'Request failed.') {
  toast.error(caught instanceof Error ? caught.message : fallback)
}

export function useToastError(message: string) {
  const shown = useRef('')

  useEffect(() => {
    if (!message) {
      shown.current = ''
      return
    }
    if (message === shown.current) return
    shown.current = message
    toast.error(message)
  }, [message])
}

export default function Toasts() {
  const dark = useSyncExternalStore(subscribeTheme, themeIsDark, () => true)

  return <Toaster theme={dark ? 'dark' : 'light'} position="bottom-right" closeButton richColors offset={16} />
}
