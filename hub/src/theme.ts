import { createTheme } from 'flowbite-react'

const field =
  'border-pink-400/30 bg-black/45 text-pink-50 placeholder:text-pink-200/40 focus:border-pink-400 focus:ring-pink-400 dark:border-pink-400/30 dark:bg-black/45 dark:text-pink-50 dark:placeholder:text-pink-200/40 dark:focus:border-pink-400 dark:focus:ring-pink-400'

export const theme = createTheme({
  alert: {
    rounded: 'rounded-2xl',
    color: {
      failure: 'border border-red-400/40 bg-red-900/50 text-red-100 dark:bg-red-900/50 dark:text-red-100',
    },
  },
  badge: {
    root: {
      color: {
        warning: 'bg-amber-300/15 text-amber-100 ring-1 ring-amber-200/40 dark:bg-amber-300/15 dark:text-amber-100',
        pink: 'bg-pink-500/20 text-pink-100 ring-1 ring-pink-300/40 dark:bg-pink-500/20 dark:text-pink-100',
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
        'bg-linear-to-r from-pink-500 to-fuchsia-600 text-white shadow-lg shadow-pink-500/30 hover:from-pink-400 hover:to-fuchsia-500 focus:ring-pink-500/40',
      light:
        'border border-pink-300/25 bg-white/5 text-pink-50 hover:border-pink-200/50 hover:bg-pink-500/15 focus:ring-pink-500/30 dark:border-pink-300/25 dark:bg-white/5 dark:text-pink-50 dark:hover:bg-pink-500/15',
      alternative:
        'border border-pink-300/20 bg-transparent text-pink-100 hover:bg-pink-500/10 focus:ring-pink-500/20 dark:border-pink-300/20 dark:bg-transparent dark:text-pink-100 dark:hover:bg-pink-500/10',
    },
  },
  card: {
    root: {
      base: 'flex rounded-3xl border border-pink-300/20 bg-linear-to-br from-white/10 via-[#3a1024]/80 to-black/75 shadow-[0_24px_70px_-36px_rgba(255,45,138,0.9)] backdrop-blur-md dark:border-pink-300/20 dark:bg-transparent',
      children: 'flex h-full flex-col justify-center gap-4 p-6',
      href: 'transition duration-300 hover:-translate-y-1 hover:border-pink-200/50 hover:shadow-[0_28px_80px_-28px_rgba(255,45,138,1)] dark:hover:bg-transparent',
    },
  },
  checkbox: {
    base: 'rounded-md border-pink-300/30 bg-black/50 dark:border-pink-300/30 dark:bg-black/50',
    color: {
      default: 'text-pink-500 focus:ring-pink-500 dark:ring-offset-black dark:focus:ring-pink-500',
    },
  },
  drawer: {
    root: {
      base: 'panel-grid fixed z-40 overflow-y-auto border border-pink-300/20 bg-[#120910]/80 p-5 shadow-[0_0_90px_-28px_rgba(255,45,138,0.85)] backdrop-blur-xl transition-transform dark:bg-[#120910]/80',
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
        gray: 'text-pink-200/65 dark:text-pink-200/65',
      },
    },
  },
  label: {
    root: {
      colors: {
        default: 'text-pink-100 dark:text-pink-100',
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
      base: 'flex items-center justify-center rounded-2xl p-2 text-base font-medium text-pink-100/80 hover:bg-pink-500/15 hover:text-white dark:text-pink-100/80 dark:hover:bg-pink-500/15',
      active: 'bg-linear-to-r from-pink-500 to-fuchsia-600 text-white shadow-md shadow-pink-500/40 dark:bg-transparent',
    },
    itemGroup: {
      base: 'mt-4 space-y-2 border-t border-pink-300/15 pt-4 first:mt-0 first:border-t-0 first:pt-0 dark:border-pink-300/15',
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
