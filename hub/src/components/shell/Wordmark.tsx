export default function Wordmark({ className = 'text-3xl' }: { className?: string }) {
  return (
    <span className={`bg-linear-to-r from-pink-800 to-pink-500 bg-clip-text font-display font-extrabold text-transparent dark:from-pink-200 dark:to-pink-500 ${className}`}>
      Bouncer
    </span>
  )
}
