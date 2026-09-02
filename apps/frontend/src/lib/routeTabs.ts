export const DEFAULT_TAB = 'orbital' as const

const VALID_TABS = [
  'orbital',
  'graph',
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
] as const

export type RouteTab = (typeof VALID_TABS)[number]

function parseHash(hash: string): RouteTab | null {
  const raw = hash.replace(/^#\/?/, '').split('?')[0].trim()
  return (VALID_TABS as readonly string[]).includes(raw) ? (raw as RouteTab) : null
}

export function getTabFromLocation(fallback: RouteTab = DEFAULT_TAB): RouteTab {
  if (typeof window === 'undefined') return fallback
  return parseHash(window.location.hash) ?? fallback
}

export function setTabInLocation(tab: RouteTab, { replace = false }: { replace?: boolean } = {}): void {
  if (typeof window === 'undefined' || !(VALID_TABS as readonly string[]).includes(tab)) return
  const next = `#/${tab}`
  if (window.location.hash === next) return
  if (replace && window.history?.replaceState) {
    const { pathname, search } = window.location
    window.history.replaceState(null, '', `${pathname}${search}${next}`)
  } else {
    window.location.hash = next
  }
}

export function subscribeTabChanges(onChange: (tab: RouteTab) => void): () => void {
  if (typeof window === 'undefined') return () => {}
  function handler() {
    const tab = parseHash(window.location.hash)
    if (tab) onChange(tab)
  }
  window.addEventListener('hashchange', handler)
  return () => window.removeEventListener('hashchange', handler)
}
