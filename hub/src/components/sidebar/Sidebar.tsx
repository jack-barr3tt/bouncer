import { useQueryClient } from '@tanstack/react-query'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { Button, Drawer, DrawerItems, Sidebar as SidebarNav, SidebarItem, SidebarItemGroup, SidebarItems } from 'flowbite-react'
import { motion } from 'motion/react'
import { useState, useSyncExternalStore, type ReactNode } from 'react'
import { HiChevronLeft, HiChevronRight } from 'react-icons/hi2'
import { getGetSessionQueryKey, getListAppsQueryKey, useLogout } from '../../api/generated.ts'
import SiteFooter from '../shell/SiteFooter.tsx'
import ThemeToggle from '../shell/ThemeToggle.tsx'
import Wordmark from '../shell/Wordmark.tsx'
import { useSession } from '../../hooks/useSession.ts'

const narrowQuery = '(max-width: 639px)'

function subscribeNarrow(onChange: () => void) {
  const query = window.matchMedia(narrowQuery)
  query.addEventListener('change', onChange)
  return () => query.removeEventListener('change', onChange)
}

export default function Sidebar({ children }: { children: ReactNode }) {
  const narrow = useSyncExternalStore(subscribeNarrow, () => window.matchMedia(narrowQuery).matches, () => false)
  const [picked, setPicked] = useState<boolean | null>(null)
  const open = picked ?? !narrow
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const logout = useLogout()
  const { session, isPending } = useSession()
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const shown = pathname !== '/login' && !pathname.startsWith('/code')

  function toggle() {
    setPicked(!open)
  }

  async function signOut() {
    await logout.mutateAsync()
    queryClient.removeQueries({ queryKey: getGetSessionQueryKey() })
    queryClient.removeQueries({ queryKey: getListAppsQueryKey() })
    await navigate({ to: '/login' })
  }

  return (
    <div className="relative z-10 min-h-svh">
      <ThemeToggle className="fixed top-4 right-4 z-50" />
      <motion.div
        aria-hidden
        className="pointer-events-none fixed -top-28 -right-16 size-120 rounded-full bg-pink-300/50 blur-3xl dark:bg-pink-500/30"
        animate={{ x: [0, -36, 0], y: [0, 28, 0] }}
        transition={{ duration: 16, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.div
        aria-hidden
        className="pointer-events-none fixed -bottom-32 -left-20 size-104 rounded-full bg-pink-200/70 blur-3xl dark:bg-fuchsia-700/25"
        animate={{ x: [0, 40, 0], y: [0, -24, 0] }}
        transition={{ duration: 18, repeat: Infinity, ease: 'easeInOut' }}
      />
      {shown && !open ? (
        <Button color="light" pill className="fixed top-4 left-4 z-50" aria-label="Expand menu" onClick={toggle}>
          <HiChevronRight className="size-5" />
        </Button>
      ) : null}
      <Drawer backdrop={false} className={shown ? undefined : 'hidden'} open={open} onClose={() => setPicked(false)}>
        <div className="mb-5 flex items-center justify-between gap-2">
          <h2>
            <Wordmark className="text-2xl font-bold" />
          </h2>
          <Button color="light" pill size="sm" aria-label="Collapse menu" onClick={toggle}>
            <HiChevronLeft className="size-5" />
          </Button>
        </div>
        <DrawerItems>
          <SidebarNav className="w-full" aria-label="Site">
            <SidebarItems>
              <SidebarItemGroup>
                <SidebarItem
                  href="/"
                  active={pathname === '/'}
                  onClick={(event) => {
                    event.preventDefault()
                    void navigate({ to: '/' })
                  }}
                >
                  Apps
                </SidebarItem>
                {!isPending && session?.role === 'admin' ? (
                  <SidebarItem
                    href="/admin"
                    active={pathname === '/admin'}
                    onClick={(event) => {
                      event.preventDefault()
                      void navigate({ to: '/admin' })
                    }}
                  >
                    Admin
                  </SidebarItem>
                ) : null}
                {!isPending && !session ? (
                  <SidebarItem
                    href="/login"
                    active={pathname === '/login'}
                    onClick={(event) => {
                      event.preventDefault()
                      void navigate({ to: '/login' })
                    }}
                  >
                    Sign in
                  </SidebarItem>
                ) : null}
                {!isPending && session ? (
                  <SidebarItem as="button" className="w-full text-left" onClick={() => void signOut()}>
                    Sign out
                  </SidebarItem>
                ) : null}
              </SidebarItemGroup>
            </SidebarItems>
          </SidebarNav>
        </DrawerItems>
      </Drawer>
      <motion.div
        key={pathname}
        className={shown ? `flex min-h-dvh flex-col ${open ? 'pl-4 sm:pl-80' : 'pl-20'}` : 'relative h-dvh'}
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
      >
        {shown ? children : <div className="h-full overflow-y-auto pb-24">{children}</div>}
        {shown ? (
          <SiteFooter />
        ) : (
          <div className="absolute inset-x-0 bottom-0">
            <SiteFooter prominent />
          </div>
        )}
      </motion.div>
    </div>
  )
}
