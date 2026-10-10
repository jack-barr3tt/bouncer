import { useQueryClient } from '@tanstack/react-query'
import { useNavigate, useRouterState } from '@tanstack/react-router'
import { Button, Drawer, DrawerItems, Sidebar, SidebarItem, SidebarItemGroup, SidebarItems } from 'flowbite-react'
import { useState, type ReactNode } from 'react'
import { HiChevronLeft, HiChevronRight } from 'react-icons/hi2'
import { getGetSessionQueryKey, getListAppsQueryKey, useLogout } from '../../api/generated.ts'
import { useSession } from '../../hooks/useSession.ts'

export default function AppShell({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(true)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const logout = useLogout()
  const { session, isPending } = useSession()
  const pathname = useRouterState({ select: (state) => state.location.pathname })

  function toggle() {
    setOpen((current) => !current)
  }

  async function signOut() {
    await logout.mutateAsync()
    queryClient.removeQueries({ queryKey: getGetSessionQueryKey() })
    queryClient.removeQueries({ queryKey: getListAppsQueryKey() })
    await navigate({ to: '/login' })
  }

  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <Button
        color="light"
        className={`fixed top-3 z-50 ${open ? 'left-74' : 'left-3'}`}
        aria-label={open ? 'Collapse menu' : 'Expand menu'}
        onClick={toggle}
      >
        {open ? <HiChevronLeft className="size-5" /> : <HiChevronRight className="size-5" />}
      </Button>
      <Drawer backdrop={false} open={open} onClose={() => setOpen(false)}>
        <h2 className="mb-4 text-base font-semibold text-gray-500">Bouncer</h2>
        <DrawerItems>
          <Sidebar className="w-full" aria-label="Site">
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
          </Sidebar>
        </DrawerItems>
      </Drawer>
      <div className={open ? 'pl-80' : 'pl-16'}>{children}</div>
    </div>
  )
}
