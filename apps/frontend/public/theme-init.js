try {
  const stored = localStorage.getItem('lyra_theme')
  const theme = stored === 'light' || stored === 'dark'
    ? stored
    : (window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark')
  if (theme === 'light') document.documentElement.classList.add('light')
} catch {
  // Theme initialization is non-critical.
}
