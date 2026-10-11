import { Checkbox, Label } from 'flowbite-react'
import { useSetAppPublic, type AppInfo } from '../../api/generated.ts'
import { useAdminAction } from '../../hooks/useAdminAction.ts'

export default function AppsSection({ apps }: { apps: AppInfo[] }) {
  const { run } = useAdminAction()
  const setAppPublic = useSetAppPublic()

  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Apps</h2>
      <p className="soft mt-2 text-sm">A public app can be opened without an account.</p>
      <ul className="mt-4 flex flex-col gap-3">
        {apps.map((app) => (
          <li key={app.slug}>
            <Label className="flex items-center gap-2">
              <Checkbox
                checked={app.public}
                onChange={(event) =>
                  void run(() => setAppPublic.mutateAsync({ slug: app.slug, data: { public: event.target.checked } }))
                }
              />
              {app.name}
            </Label>
          </li>
        ))}
      </ul>
    </section>
  )
}
