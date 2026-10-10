import { createFileRoute } from '@tanstack/react-router'
import CodeForm from '../components/login/CodeForm.tsx'
import SignInForm from '../components/login/SignInForm.tsx'
import Fade from '../components/motion/Fade.tsx'

export const Route = createFileRoute('/login')({
  component: Login,
})

function Login() {
  return (
    <div className="min-h-svh">
      <main className="mx-auto flex max-w-md flex-col gap-8 px-4 py-16">
        <Fade>
          <header>
            <p className="text-xs font-semibold tracking-[0.22em] text-pink-300 uppercase">Personal</p>
            <h1 className="mt-2 text-5xl font-extrabold">Sign in</h1>
          </header>
        </Fade>
        <Fade delay={0.08}>
          <SignInForm />
        </Fade>
        <Fade delay={0.16}>
          <CodeForm />
        </Fade>
      </main>
    </div>
  )
}
