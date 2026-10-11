import { format, parseISO } from 'date-fns'
import { Button, Card, Label, Select, TextInput } from 'flowbite-react'
import { useState, type SubmitEvent } from 'react'
import { useAddRepo, useGetDeploy, usePreviewRepo, useRunDeploy } from '../../api/generated.ts'
import { useAdminAction } from '../../hooks/useAdminAction.ts'
import { reportError, useToastError } from '../../toasts.tsx'

const shownTime = 'd MMM yyyy, HH:mm'

type DetectedApp = {
  dir: string
  slug: string
  name: string
  description: string
  icon: string
}

function displayName(slug: string): string {
  return slug.replace(/(^|-)([a-z])/g, (_, gap: string, letter: string) => `${gap ? ' ' : ''}${letter.toUpperCase()}`)
}

export default function DeploySection() {
  const { run } = useAdminAction()
  const deployQuery = useGetDeploy({
    query: {
      refetchInterval: (query) => (query.state.data?.repos.some((repo) => repo.latest?.status === 'running') ? 2000 : false),
    },
  })
  const deploy = deployQuery.data
  const [remote, setRemote] = useState('')
  const [name, setName] = useState('')
  const [mode, setMode] = useState<'branch' | 'commit'>('branch')
  const [ref, setRef] = useState('main')
  const [detected, setDetected] = useState<DetectedApp[]>([])
  const runDeploy = useRunDeploy()
  const previewRepo = usePreviewRepo()
  const addRepo = useAddRepo()
  useToastError(deployQuery.error instanceof Error ? deployQuery.error.message : '')

  async function onPreview(event: SubmitEvent) {
    event.preventDefault()
    try {
      const result = await previewRepo.mutateAsync({
        data: { remote, name, branch: mode === 'branch' ? ref : undefined, commit: mode === 'commit' ? ref : undefined },
      })
      setDetected(
        result.apps.map((app) => {
          const slug = (app.dir === '.' ? name : app.dir.split('/').filter(Boolean).pop()) ?? name
          const label = displayName(slug)
          return { dir: app.dir, slug, name: label, description: '', icon: label.slice(0, 1) || 'A' }
        }),
      )
    } catch (caught) {
      reportError(caught)
    }
  }

  async function onAdd(event: SubmitEvent) {
    event.preventDefault()
    await run(async () => {
      await addRepo.mutateAsync({
        data: {
          remote,
          name,
          branch: mode === 'branch' ? ref : undefined,
          commit: mode === 'commit' ? ref : undefined,
          apps: detected.map((app) => ({
            slug: app.slug,
            name: app.name,
            description: app.description,
            icon: app.icon,
            dir: app.dir,
          })),
        },
      })
      setDetected([])
      setRemote('')
      setName('')
      setMode('branch')
      setRef('main')
      await deployQuery.refetch()
    })
  }

  function updateDetected(index: number, field: 'slug' | 'name' | 'description' | 'icon', value: string) {
    setDetected((current) => current.map((app, item) => (item === index ? { ...app, [field]: value } : app)))
  }

  return (
    <section className="mt-10">
      <h2 className="text-xl font-semibold">Deploy</h2>
      <form className="mt-4 flex flex-wrap items-end gap-3" onSubmit={(event) => void onPreview(event)}>
        <div>
          <Label htmlFor="repo-remote">Git remote</Label>
          <TextInput
            id="repo-remote"
            className="mt-1"
            value={remote}
            onChange={(event) => setRemote(event.target.value)}
            placeholder="git@github.com:you/notes.git"
            required
          />
        </div>
        <div>
          <Label htmlFor="repo-name">Name</Label>
          <TextInput
            id="repo-name"
            className="mt-1"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="notes"
            required
          />
        </div>
        <div>
          <Label htmlFor="repo-mode">Publish</Label>
          <Select
            id="repo-mode"
            className="mt-1"
            value={mode}
            onChange={(event) => {
              const next = event.target.value === 'commit' ? 'commit' : 'branch'
              setMode(next)
              setRef(next === 'branch' ? 'main' : '')
            }}
          >
            <option value="branch">Track a branch</option>
            <option value="commit">Pin a commit</option>
          </Select>
        </div>
        <div>
          <Label htmlFor="repo-ref">{mode === 'branch' ? 'Branch' : 'Commit'}</Label>
          <TextInput
            id="repo-ref"
            className="mt-1"
            value={ref}
            onChange={(event) => setRef(event.target.value)}
            required
          />
        </div>
        <Button type="submit">Find apps</Button>
      </form>
      {detected.length > 0 ? (
        <form className="mt-4 grid gap-4" onSubmit={(event) => void onAdd(event)}>
          {detected.map((app, index) => (
            <Card key={app.dir}>
              <div className="flex flex-col gap-4">
                <div className="flex items-center justify-between gap-3">
                  <p className="font-mono text-sm">{app.dir}</p>
                  <Button type="button" color="light" size="sm" onClick={() => setDetected((current) => current.filter((_, item) => item !== index))}>
                    Remove
                  </Button>
                </div>
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
                </div>
              </div>
            </Card>
          ))}
          <Button type="submit" className="w-fit">
            Add repository
          </Button>
        </form>
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
                  {repo.commit ? ` · ${repo.commit.slice(0, 7)}` : ''}
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
                      <pre className="soft mt-2 max-h-48 overflow-auto rounded-2xl border border-pink-200 bg-white/70 p-3 whitespace-pre-wrap text-xs dark:border-pink-300/15 dark:bg-black/40">
                        {repo.latest.log}
                      </pre>
                    ) : null}
                  </div>
                ) : (
                  <p className="soft text-sm">No deploys yet.</p>
                )}
              </Card>
            ))}
          </div>
        </div>
      ) : (
        <p className="soft mt-4 text-sm">Bouncer is serving the files already in the site directory. It fetches and builds from git after you add a repository here, or set GIT_REMOTE for this site.</p>
      )}
    </section>
  )
}
