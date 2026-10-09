import { format, parseISO } from 'date-fns'
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
        <p className="mt-4 text-sm text-red-700" role="alert">
          {shownError}
        </p>
      ) : null}
      <form className="mt-4 flex flex-wrap items-end gap-3" onSubmit={(event) => void onClone(event)}>
        <label className="text-sm">
          Git remote
          <input
            className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
            value={cloneRemote}
            onChange={(event) => setCloneRemote(event.target.value)}
            placeholder="git@github.com:you/notes.git"
            required
          />
        </label>
        <label className="text-sm">
          Folder
          <input
            className="mt-1 block w-32 rounded-xl border border-stone-300 px-3 py-2"
            value={cloneName}
            onChange={(event) => setCloneName(event.target.value)}
            placeholder="notes"
            required
          />
        </label>
        <label className="text-sm">
          Branch
          <input
            className="mt-1 block w-28 rounded-xl border border-stone-300 px-3 py-2"
            value={cloneBranch}
            onChange={(event) => setCloneBranch(event.target.value)}
            required
          />
        </label>
        <button className="rounded-full bg-stone-900 px-4 py-2 text-sm text-white" type="submit">
          Clone into apps/
        </button>
      </form>
      {detected.length > 0 ? (
        <div className="mt-4 grid gap-4">
          {detected.map((app, index) => (
            <form
              className="rounded-2xl border border-stone-200 bg-white p-4"
              key={app.source}
              onSubmit={(event) => {
                event.preventDefault()
                void onAddApp(index)
              }}
            >
              <p className="font-mono text-sm">{app.source}</p>
              <div className="mt-3 flex flex-wrap items-end gap-3">
                <label className="text-sm">
                  Slug
                  <input
                    className="mt-1 block w-28 rounded-xl border border-stone-300 px-3 py-2"
                    value={app.slug}
                    onChange={(event) => updateDetected(index, 'slug', event.target.value)}
                    required
                  />
                </label>
                <label className="text-sm">
                  Name
                  <input
                    className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                    value={app.name}
                    onChange={(event) => updateDetected(index, 'name', event.target.value)}
                    required
                  />
                </label>
                <label className="text-sm">
                  Description
                  <input
                    className="mt-1 block rounded-xl border border-stone-300 px-3 py-2"
                    value={app.description}
                    onChange={(event) => updateDetected(index, 'description', event.target.value)}
                    required
                  />
                </label>
                <label className="text-sm">
                  Icon
                  <input
                    className="mt-1 block w-20 rounded-xl border border-stone-300 px-3 py-2"
                    value={app.icon}
                    onChange={(event) => updateDetected(index, 'icon', event.target.value)}
                    required
                  />
                </label>
                <button className="rounded-full bg-stone-900 px-4 py-2 text-sm text-white" type="submit">
                  Add app
                </button>
              </div>
            </form>
          ))}
        </div>
      ) : null}
      {deploy == null ? null : deploy.enabled ? (
        <div className="mt-4">
          <p className="font-mono text-sm">{deploy.webhookUrl}</p>
          <button className="mt-3 text-sm underline" type="button" onClick={() => void navigator.clipboard.writeText(deploy.webhookUrl)}>
            Copy webhook URL
          </button>
          <div className="mt-4 grid gap-4">
            {deploy.repos.map((repo) => (
              <div className="rounded-2xl border border-stone-200 bg-white p-4" key={`${repo.path}:${repo.remote}`}>
                <p className="text-sm">
                  {repo.path ? `${repo.path} · ` : ''}
                  {repo.remote}
                  {repo.branch ? ` · ${repo.branch}` : ''}
                </p>
                <div className="mt-3 text-sm">
                  <button
                    className="underline disabled:text-stone-400"
                    type="button"
                    disabled={repo.latest?.status === 'running'}
                    onClick={() => void run(() => runDeploy.mutateAsync({ data: { remote: repo.remote } }))}
                  >
                    Deploy now
                  </button>
                </div>
                {repo.latest ? (
                  <div className="mt-4">
                    <p className="text-sm">
                      {repo.latest.status} · {repo.latest.sha.slice(0, 7)}
                      {repo.latest.finishedAt ? ` · ${format(parseISO(repo.latest.finishedAt), shownTime)}` : ''}
                    </p>
                    {repo.latest.log ? (
                      <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap text-xs text-stone-600">{repo.latest.log}</pre>
                    ) : null}
                  </div>
                ) : (
                  <p className="mt-4 text-sm text-stone-600">No deploys yet.</p>
                )}
              </div>
            ))}
          </div>
        </div>
      ) : (
        <p className="mt-4 text-sm text-stone-600">Git deploy is off until a clone under apps/ is registered, or GIT_REMOTE is set.</p>
      )}
    </section>
  )
}
