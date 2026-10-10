import { format, parseISO } from 'date-fns'
import { Alert, Button, Card, Label, TextInput } from 'flowbite-react'
import { useState, type SubmitEvent } from 'react'
import { useCloneRepo, useCreateApp, useGetDeploy, useRunDeploy } from '../../api/generated.ts'
import { useAdminAction } from '../../hooks/useAdminAction.ts'

const shownTime = 'd MMM yyyy, HH:mm'

type DetectedApp = {
  source: string
  slug: string
  name: string
  description: string
  icon: string
}

export default function DeploySection() {
  const { error, setError, run } = useAdminAction()
  const deployQuery = useGetDeploy({
    query: {
      refetchInterval: (query) => (query.state.data?.repos.some((repo) => repo.latest?.status === 'running') ? 2000 : false),
    },
  })
  const deploy = deployQuery.data
  const [cloneRemote, setCloneRemote] = useState('')
  const [cloneName, setCloneName] = useState('')
  const [cloneBranch, setCloneBranch] = useState('main')
  const [detected, setDetected] = useState<DetectedApp[]>([])
  const runDeploy = useRunDeploy()
  const cloneRepo = useCloneRepo()
  const createApp = useCreateApp()
  const loadError = deployQuery.error instanceof Error ? deployQuery.error.message : ''
  const shownError = error || loadError

  async function onClone(event: SubmitEvent) {
    event.preventDefault()
    setError('')
    try {
      const result = await cloneRepo.mutateAsync({ data: { remote: cloneRemote, name: cloneName, branch: cloneBranch } })
      setDetected(
        result.apps.map((app) => {
          const slug = app.source.split('/').filter(Boolean).pop() ?? ''
          const name = slug.replace(/(^|-)([a-z])/g, (_, gap: string, letter: string) => `${gap ? ' ' : ''}${letter.toUpperCase()}`)
          return { source: app.source, slug, name, description: '', icon: name.slice(0, 1) || 'A' }
        }),
      )
      setCloneRemote('')
      setCloneName('')
      setCloneBranch('main')
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Request failed.')
    }
  }

  async function onAddApp(index: number) {
    const app = detected[index]
    if (!app) return
    await run(async () => {
      await createApp.mutateAsync({ data: app })
      setDetected((current) => current.filter((_, item) => item !== index))
    })
  }

  function updateDetected(index: number, field: 'slug' | 'name' | 'description' | 'icon', value: string) {
    setDetected((current) => current.map((app, item) => (item === index ? { ...app, [field]: value } : app)))
  }

  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Deploy</h2>
      {shownError ? (
        <Alert className="mt-4" color="failure">
          {shownError}
        </Alert>
      ) : null}
      <form className="mt-4 flex flex-wrap items-end gap-3" onSubmit={(event) => void onClone(event)}>
        <div>
          <Label htmlFor="clone-remote">Git remote</Label>
          <TextInput
            id="clone-remote"
            className="mt-1"
            value={cloneRemote}
            onChange={(event) => setCloneRemote(event.target.value)}
            placeholder="git@github.com:you/notes.git"
            required
          />
        </div>
        <div>
          <Label htmlFor="clone-folder">Folder</Label>
          <TextInput
            id="clone-folder"
            className="mt-1"
            value={cloneName}
            onChange={(event) => setCloneName(event.target.value)}
            placeholder="notes"
            required
          />
        </div>
        <div>
          <Label htmlFor="clone-branch">Branch</Label>
          <TextInput
            id="clone-branch"
            className="mt-1"
            value={cloneBranch}
            onChange={(event) => setCloneBranch(event.target.value)}
            required
          />
        </div>
        <Button type="submit">Clone into apps/</Button>
      </form>
      {detected.length > 0 ? (
        <div className="mt-4 grid gap-4">
          {detected.map((app, index) => (
            <Card key={app.source}>
              <form
                className="flex flex-col gap-4"
                onSubmit={(event) => {
                  event.preventDefault()
                  void onAddApp(index)
                }}
              >
                <p className="font-mono text-sm">{app.source}</p>
                <div className="flex flex-wrap items-end gap-3">
                  <div>
                    <Label htmlFor={`app-slug-${index}`}>Slug</Label>
                    <TextInput
                      id={`app-slug-${index}`}
                      className="mt-1"
                      value={app.slug}
                      onChange={(event) => updateDetected(index, 'slug', event.target.value)}
                      required
                    />
                  </div>
                  <div>
                    <Label htmlFor={`app-name-${index}`}>Name</Label>
                    <TextInput
                      id={`app-name-${index}`}
                      className="mt-1"
                      value={app.name}
                      onChange={(event) => updateDetected(index, 'name', event.target.value)}
                      required
                    />
                  </div>
                  <div>
                    <Label htmlFor={`app-description-${index}`}>Description</Label>
                    <TextInput
                      id={`app-description-${index}`}
                      className="mt-1"
                      value={app.description}
                      onChange={(event) => updateDetected(index, 'description', event.target.value)}
                      required
                    />
                  </div>
                  <div>
                    <Label htmlFor={`app-icon-${index}`}>Icon</Label>
                    <TextInput
                      id={`app-icon-${index}`}
                      className="mt-1"
                      value={app.icon}
                      onChange={(event) => updateDetected(index, 'icon', event.target.value)}
                      required
                    />
                  </div>
                  <Button type="submit">Add app</Button>
                </div>
              </form>
            </Card>
          ))}
        </div>
      ) : null}
      {deploy == null ? null : deploy.enabled ? (
        <div className="mt-4">
          <p className="font-mono text-sm">{deploy.webhookUrl}</p>
          <Button className="mt-3" color="light" size="sm" onClick={() => void navigator.clipboard.writeText(deploy.webhookUrl)}>
            Copy webhook URL
          </Button>
          <div className="mt-4 grid gap-4">
            {deploy.repos.map((repo) => (
              <Card key={`${repo.path}:${repo.remote}`}>
                <p className="text-sm">
                  {repo.path ? `${repo.path} · ` : ''}
                  {repo.remote}
                  {repo.branch ? ` · ${repo.branch}` : ''}
                </p>
                <Button
                  color="light"
                  size="sm"
                  className="w-fit"
                  disabled={repo.latest?.status === 'running'}
                  onClick={() => void run(() => runDeploy.mutateAsync({ data: { remote: repo.remote } }))}
                >
                  Deploy now
                </Button>
                {repo.latest ? (
                  <div>
                    <p className="text-sm">
                      {repo.latest.status} · {repo.latest.sha.slice(0, 7)}
                      {repo.latest.finishedAt ? ` · ${format(parseISO(repo.latest.finishedAt), shownTime)}` : ''}
                    </p>
                    {repo.latest.log ? (
                      <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap text-xs text-stone-600">{repo.latest.log}</pre>
                    ) : null}
                  </div>
                ) : (
                  <p className="text-sm text-stone-600">No deploys yet.</p>
                )}
              </Card>
            ))}
          </div>
        </div>
      ) : (
        <p className="mt-4 text-sm text-stone-600">Git deploy is off until a clone under apps/ is registered, or GIT_REMOTE is set.</p>
      )}
    </section>
  )
}
