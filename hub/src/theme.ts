import { createTheme } from 'flowbite-react'

const field =
  'border-pink-200 bg-white/80 text-pink-900 placeholder:text-pink-800/55 focus:border-pink-500 focus:ring-pink-400 dark:border-pink-400/30 dark:bg-black/45 dark:text-pink-50 dark:placeholder:text-pink-200/40 dark:focus:border-pink-400 dark:focus:ring-pink-400'

export const theme = createTheme({
  alert: {
    rounded: 'rounded-2xl',
    color: {
      failure: 'border border-red-200 bg-red-50 text-red-800 dark:border-red-400/40 dark:bg-red-900/50 dark:text-red-100',
    },
  },
  badge: {
    root: {
      color: {
        warning: 'bg-amber-100 text-amber-900 ring-1 ring-amber-300 dark:bg-amber-300/15 dark:text-amber-100 dark:ring-amber-200/40',
        pink: 'bg-pink-100 text-pink-800 ring-1 ring-pink-300 dark:bg-pink-500/20 dark:text-pink-100 dark:ring-pink-300/40',
      },
    },
    icon: {
      off: 'rounded-full px-2.5 py-1',
    },
  },
  button: {
    base: 'rounded-2xl transition duration-200',
    color: {
      default:
        'bg-linear-to-r from-pink-500 to-fuchsia-600 text-white shadow-lg shadow-pink-500/25 hover:from-pink-400 hover:to-fuchsia-500 focus:ring-pink-500/40 dark:shadow-pink-500/30',
      light:
        'border border-pink-200 bg-white/80 text-pink-900 hover:border-pink-300 hover:bg-pink-50 focus:ring-pink-400/40 dark:border-pink-300/25 dark:bg-white/5 dark:text-pink-50 dark:hover:border-pink-200/50 dark:hover:bg-pink-500/15 dark:focus:ring-pink-500/30',
      alternative:
        'border border-pink-200 bg-white/40 text-pink-800 hover:bg-pink-50 focus:ring-pink-400/30 dark:border-pink-300/20 dark:bg-transparent dark:text-pink-100 dark:hover:bg-pink-500/10 dark:focus:ring-pink-500/20',
    },
  },
  card: {
    root: {
      base: 'flex rounded-3xl border border-pink-200/90 bg-linear-to-br from-white via-[#fff5f8] to-pink-100/80 text-inherit shadow-[0_24px_60px_-36px_rgba(208,40,110,0.4)] backdrop-blur-md dark:border-pink-300/20 dark:bg-transparent dark:from-white/10 dark:via-[#3a1024]/80 dark:to-black/75 dark:text-inherit dark:shadow-[0_24px_70px_-36px_rgba(255,45,138,0.9)]',
      children: 'flex h-full flex-col justify-center gap-4 p-6',
      href: 'transition duration-300 hover:-translate-y-1 hover:border-pink-300 hover:shadow-[0_28px_70px_-28px_rgba(208,40,110,0.45)] dark:hover:border-pink-200/50 dark:hover:bg-transparent dark:hover:shadow-[0_28px_80px_-28px_rgba(255,45,138,1)]',
    },
  },
  checkbox: {
    base: 'rounded-md border-pink-300 bg-white dark:border-pink-300/30 dark:bg-black/50',
    color: {
      default: 'text-pink-600 focus:ring-pink-500 dark:text-pink-500 dark:ring-offset-black dark:focus:ring-pink-500',
    },
  },
  drawer: {
    root: {
      base: 'panel-grid fixed z-40 overflow-y-auto border border-pink-200/80 bg-white/75 p-5 shadow-[0_0_80px_-32px_rgba(208,40,110,0.45)] backdrop-blur-xl transition-transform dark:border-pink-300/20 dark:bg-[#120910]/80 dark:shadow-[0_0_90px_-28px_rgba(255,45,138,0.85)]',
      position: {
        left: {
          on: 'top-3 left-3 h-[calc(100svh-1.5rem)] w-72 transform-none rounded-[1.75rem]',
          off: 'top-3 left-3 h-[calc(100svh-1.5rem)] w-72 -translate-x-[120%] rounded-[1.75rem]',
        },
      },
    },
  },
  helperText: {
    root: {
      colors: {
        gray: 'text-pink-800/65 dark:text-pink-200/65',
      },
    },
  },
  label: {
    root: {
      colors: {
        default: 'text-pink-900 dark:text-pink-100',
      },
    },
  },
  select: {
    field: {
      select: {
        withAddon: { off: 'rounded-2xl' },
        colors: { gray: field },
      },
    },
  },
  sidebar: {
    root: {
      inner: 'h-full overflow-y-auto overflow-x-hidden bg-transparent px-1 py-2 dark:bg-transparent',
    },
    item: {
      base: 'flex items-center justify-center rounded-2xl p-2 text-base font-medium text-pink-900/75 hover:bg-pink-500/10 hover:text-pink-900 dark:text-pink-100/80 dark:hover:bg-pink-500/15 dark:hover:text-white',
      active: 'bg-linear-to-r from-pink-500 to-fuchsia-600 text-white shadow-md shadow-pink-500/30 dark:bg-transparent dark:shadow-pink-500/40',
    },
    itemGroup: {
      base: 'mt-4 space-y-2 border-t border-pink-200 pt-4 first:mt-0 first:border-t-0 first:pt-0 dark:border-pink-300/15',
    },
  },
  textInput: {
    field: {
      input: {
        withAddon: { off: 'rounded-2xl' },
        colors: { gray: field },
      },
    },
  },
})
