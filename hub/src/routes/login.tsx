import { createFileRoute } from '@tanstack/react-router'
import CodeForm from '../components/login/CodeForm.tsx'
import SignInForm from '../components/login/SignInForm.tsx'
import Fade from '../components/motion/Fade.tsx'
import Wordmark from '../components/shell/Wordmark.tsx'

export const Route = createFileRoute('/login')({
  component: Login,
})

function Login() {
  return (
    <div>
      <main className="mx-auto flex max-w-md flex-col gap-8 px-4 py-16 lg:max-w-4xl">
        <Fade>
          <header>
            <Wordmark />
            <h1 className="mt-8 text-5xl font-extrabold">Sign in</h1>
          </header>
        </Fade>
        <div className="grid items-stretch gap-8 lg:grid-cols-2">
          <Fade className="h-full" delay={0.08}>
            <SignInForm />
          </Fade>
          <Fade className="h-full" delay={0.16}>
            <CodeForm />
          </Fade>
        </div>
      </main>
    </div>
  )
}
