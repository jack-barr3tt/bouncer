import { Checkbox, Label } from 'flowbite-react'
import { useGetSettings, useUpdateSettings } from '../../api/generated.ts'
import { useAdminAction } from '../../hooks/useAdminAction.ts'
import { useToastError } from '../../toasts.tsx'

export default function SettingsSection() {
  const { run } = useAdminAction()
  const settings = useGetSettings()
  const updateSettings = useUpdateSettings()
  useToastError(settings.error instanceof Error ? settings.error.message : '')

  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Settings</h2>
      {settings.data ? (
        <Label className="mt-4 flex items-center gap-2">
          <Checkbox
            checked={settings.data.autoUpdate}
            disabled={updateSettings.isPending}
            onChange={(event) => void run(() => updateSettings.mutateAsync({ data: { autoUpdate: event.target.checked } }))}
          />
          Update Bouncer automatically
        </Label>
      ) : null}
    </section>
  )
}
