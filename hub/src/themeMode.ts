const key = 'bouncer-theme'

function emit() {
  window.dispatchEvent(new Event('bouncer-theme'))
}

export function subscribeTheme(onChange: () => void) {
  window.addEventListener('bouncer-theme', onChange)
  return () => window.removeEventListener('bouncer-theme', onChange)
}

export function themeIsDark() {
  return document.documentElement.classList.contains('dark')
}

export function setTheme(dark: boolean) {
  document.documentElement.classList.toggle('dark', dark)
  localStorage.setItem(key, dark ? 'dark' : 'light')
  emit()
}
