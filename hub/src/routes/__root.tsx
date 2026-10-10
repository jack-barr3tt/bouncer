import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { HeadContent, Outlet, Scripts, createRootRoute } from '@tanstack/react-router'
import { ThemeProvider } from 'flowbite-react'
import { MotionConfig } from 'motion/react'
import { subscribeTheme, themeIsDark } from '../themeMode.ts'
import { theme } from '../theme.ts'
import { useState, useSyncExternalStore, type ReactNode } from 'react'
import { ThemeInit } from '../../.flowbite-react/init.tsx'
import Sidebar from '../components/sidebar/Sidebar.tsx'
import { useSession, useWatchAccess } from '../hooks/useSession.ts'
import appCss from '../index.css?url'

export const Route = createRootRoute({
  head: () => ({
    meta: [
      { charSet: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      { title: 'Apps' },
    ],
    links: [
      { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' },
      { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
      { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossOrigin: 'anonymous' },
      {
        rel: 'stylesheet',
        href: 'https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700&family=Syne:wght@600;700;800&display=swap',
      },
      { rel: 'stylesheet', href: appCss },
    ],
  }),
  component: RootComponent,
})

function RootComponent() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: 1, refetchOnWindowFocus: false },
        },
      }),
  )

  return (
    <RootDocument>
      <QueryClientProvider client={queryClient}>
        <AccessWatch />
        <Sidebar>
          <Outlet />
        </Sidebar>
      </QueryClientProvider>
    </RootDocument>
  )
}

function AccessWatch() {
  const { session } = useSession()
  useWatchAccess(session)
  return null
}

const themeScript = `(function(){try{if(localStorage.getItem('bouncer-theme')==='light')document.documentElement.classList.remove('dark')}catch(e){}})()`

function RootDocument({ children }: Readonly<{ children: ReactNode }>) {
  const dark = useSyncExternalStore(subscribeTheme, themeIsDark, () => true)

  return (
    <html lang="en" className={dark ? 'dark' : undefined} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
        <HeadContent />
      </head>
      <body suppressHydrationWarning>
        <ThemeInit />
        <ThemeProvider theme={theme}>
          <MotionConfig reducedMotion="user">{children}</MotionConfig>
        </ThemeProvider>
        <Scripts />
      </body>
    </html>
  )
}
