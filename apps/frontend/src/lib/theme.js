const STORAGE_KEY = 'lyra_theme'

export function getStoredTheme() {
  const stored = localStorage.getItem(STORAGE_KEY)
  return stored === 'light' || stored === 'dark' ? stored : null
}

export function getSystemTheme() {
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

export function getPreferredTheme() {
  return getStoredTheme() ?? getSystemTheme()
}

export function applyTheme(theme) {
  document.documentElement.classList.toggle('light', theme === 'light')
}

export function setTheme(theme) {
  localStorage.setItem(STORAGE_KEY, theme)
  applyTheme(theme)
}
