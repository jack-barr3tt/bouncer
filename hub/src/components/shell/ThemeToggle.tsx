import { Button } from 'flowbite-react'
import { useSyncExternalStore } from 'react'
import { HiMoon, HiSun } from 'react-icons/hi2'
import { setTheme, subscribeTheme, themeIsDark } from '../../themeMode.ts'

export default function ThemeToggle({ className = '' }: { className?: string }) {
  const dark = useSyncExternalStore(subscribeTheme, themeIsDark, () => true)

  return (
    <Button
      color="light"
      pill
      size="sm"
      className={className}
      aria-label={dark ? 'Switch to light mode' : 'Switch to dark mode'}
      onClick={() => setTheme(!dark)}
    >
      {dark ? <HiSun className="size-5" /> : <HiMoon className="size-5" />}
    </Button>
  )
}
