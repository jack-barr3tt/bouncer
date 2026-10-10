import { useQueryClient } from '@tanstack/react-query'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { Button, Drawer, DrawerItems, Sidebar as SidebarNav, SidebarItem, SidebarItemGroup, SidebarItems } from 'flowbite-react'
import { motion } from 'motion/react'
import { useState, useSyncExternalStore, type ReactNode } from 'react'
import { HiChevronLeft, HiChevronRight } from 'react-icons/hi2'
import { getGetSessionQueryKey, getListAppsQueryKey, useLogout } from '../../api/generated.ts'
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
    <div className="relative z-10 min-h-svh text-pink-50">
      <motion.div
        aria-hidden
        className="pointer-events-none fixed -top-28 -right-16 size-[30rem] rounded-full bg-pink-500/30 blur-3xl"
        animate={{ x: [0, -36, 0], y: [0, 28, 0] }}
        transition={{ duration: 16, repeat: Infinity, ease: 'easeInOut' }}
      />
      <motion.div
        aria-hidden
        className="pointer-events-none fixed -bottom-32 -left-20 size-[26rem] rounded-full bg-fuchsia-700/25 blur-3xl"
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
          <h2 className="bg-linear-to-r from-pink-200 to-pink-500 bg-clip-text text-2xl font-bold text-transparent">Bouncer</h2>
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
                  <SidebarItem as="button" onClick={() => void signOut()}>
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
        className={shown ? (open ? 'pl-4 sm:pl-80' : 'pl-20') : undefined}
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
      >
        {children}
      </motion.div>
    </div>
  )
}
