import { SiGithub } from 'react-icons/si'

const repo = 'https://github.com/jack-barr3tt/bouncer'
const year = new Date().getFullYear()

export default function SiteFooter({ prominent = false }: { prominent?: boolean }) {

  return (
    <footer
      className={
        prominent
          ? 'flex shrink-0 justify-center bg-transparent px-4 pt-2 pb-4'
          : 'mt-auto flex items-center justify-center gap-3 px-4 py-6 text-xs text-pink-200/50'
      }
    >
      <div
        className={
          prominent
            ? 'flex items-center gap-3 rounded-full border border-pink-300/40 bg-[#160910]/85 px-5 py-2 text-sm text-pink-50 shadow-[0_10px_30px_-14px_rgba(255,45,138,0.9)] backdrop-blur-md'
            : 'contents'
        }
      >
        <p>© Jack Barrett {year}</p>
        <a
          className={prominent ? 'text-pink-50 hover:text-white' : 'hover:text-pink-100'}
          href={repo}
          target="_blank"
          rel="noreferrer"
          aria-label="Bouncer on GitHub"
        >
          <SiGithub className={prominent ? 'size-5' : 'size-4'} />
        </a>
      </div>
    </footer>
  )
}
