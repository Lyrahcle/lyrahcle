
export const DEFAULT_TAB = 'orbital'

const VALID_TABS = [
  'orbital',
  'stations',
  'assets',
  'signal',
  'incidents',
  'intel',
  'decoder',
  'connectivity',
  'admin',
  'platform',
  'settings',
]

function parseHash(hash) {
  const raw = hash.replace(/^#\/?/, '').split('?')[0].trim()
  return VALID_TABS.includes(raw) ? raw : null
}

export function getTabFromLocation(fallback = DEFAULT_TAB) {
  if (typeof window === 'undefined') return fallback
  return parseHash(window.location.hash) ?? fallback
}

export function setTabInLocation(tab, { replace = false } = {}) {
  if (typeof window === 'undefined' || !VALID_TABS.includes(tab)) return
  const next = `#/${tab}`
  if (window.location.hash === next) return
  if (replace && window.history?.replaceState) {
    const { pathname, search } = window.location
    window.history.replaceState(null, '', `${pathname}${search}${next}`)
  } else {
    window.location.hash = next
  }
}

export function subscribeTabChanges(onChange) {
  if (typeof window === 'undefined') return () => {}
  function handler() {
    const tab = parseHash(window.location.hash)
    if (tab) onChange(tab)
  }
  window.addEventListener('hashchange', handler)
  return () => window.removeEventListener('hashchange', handler)
}
