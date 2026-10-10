import { createFileRoute } from '@tanstack/react-router'
import CodeForm from '../components/login/CodeForm.tsx'
import SignInForm from '../components/login/SignInForm.tsx'

export const Route = createFileRoute('/login')({
  component: Login,
})

function Login() {
  return (
    <div className="min-h-svh bg-stone-100 text-stone-900">
      <main className="mx-auto flex max-w-md flex-col gap-8 px-4 py-16">
        <header>
          <p className="text-sm font-medium text-stone-500">Personal</p>
          <h1 className="mt-2 text-4xl font-semibold tracking-tight">Sign in</h1>
        </header>
        <SignInForm />
        <CodeForm />
      </main>
    </div>
  )
}
