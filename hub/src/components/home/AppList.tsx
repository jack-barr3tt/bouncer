import { Link } from '@tanstack/react-router'
import { Badge, Card } from 'flowbite-react'
import { motion } from 'motion/react'
import { useAppAvailability } from '../../hooks/useAppAvailability.ts'
import { useApps, useSession } from '../../hooks/useSession.ts'

const list = {
  hidden: {},
  show: { transition: { staggerChildren: 0.07 } },
}

const item = {
  hidden: { opacity: 0, y: 18 },
  show: { opacity: 1, y: 0, transition: { type: 'spring' as const, stiffness: 320, damping: 26 } },
}

export default function AppList() {
  const { session } = useSession()
  const { apps } = useApps()
  const availability = useAppAvailability(session, apps)

  return (
    <section className="mt-8">
      {!session ? (
        <Card className="max-w-xl">
          <p className="font-medium">Sign in to see your apps</p>
          <Link className="accent text-sm font-medium underline" to="/login">
            Sign in
          </Link>
        </Card>
      ) : null}

      {session && apps.length === 0 ? (
        <Card className="max-w-xl">
          <p className="font-medium">No apps yet</p>
          <p className="soft text-sm">Nothing has been turned on for this account.</p>
        </Card>
      ) : null}

      {session && apps.length > 0 ? (
        <motion.ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" initial="hidden" animate="show" variants={list}>
          {apps.map((app) => {
            const down = availability[app.slug] === 'down'
            return (
              <motion.li key={app.slug} variants={item} whileHover={{ y: -6 }}>
                <Card className="h-full" href={app.path}>
                  <span className="flex size-12 items-center justify-center rounded-2xl bg-linear-to-br from-pink-400 to-fuchsia-700 text-sm font-semibold text-white shadow-lg shadow-pink-500/40">
                    {app.icon}
                  </span>
                  <span className="font-display text-lg font-bold wrap-break-word">{app.name}</span>
                  <span className="soft text-sm leading-6">{app.description}</span>
                  {down ? (
                    <Badge className="w-fit" color="warning">
                      Unavailable
                    </Badge>
                  ) : null}
                </Card>
              </motion.li>
            )
          })}
        </motion.ul>
      ) : null}
    </section>
  )
}
