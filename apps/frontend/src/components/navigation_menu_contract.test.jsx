import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

function source(relativePath) {
  return readFileSync(resolve(process.cwd(), 'src', relativePath), 'utf8')
}

describe('navigation shell contract', () => {
  it('uses a persistent sidebar instead of horizontal primary tabs', () => {
    const dashboard = source('components/Dashboard.tsx')
    const header = source('components/Header.jsx')
    expect(dashboard).not.toContain("import Tabs from './Tabs.jsx'")
    expect(dashboard).toContain('<NavigationMenu')
    expect(dashboard).toContain('lg:pl-20')
    expect(dashboard).toContain('lg:pl-64')
    expect(header).toContain('onOpenNavigation')
  })

  it('keeps orbital as the home route and settings in typed page navigation', () => {
    const dashboard = source('components/Dashboard.tsx')
    const navigation = source('components/dashboard/navigation.ts')
    const routeTabs = source('lib/routeTabs.ts')
    expect(routeTabs).toContain("export const DEFAULT_TAB = 'orbital'")
    expect(dashboard).toContain("onHome={() => onChangeTab('orbital')}")
    expect(navigation).toContain("{ id: 'settings', label: t('tabs.settings'), section: 'preferences' }")
  })

  it('keeps URL hash routing isolated in the typed route hook', () => {
    const app = source('App.tsx')
    const hook = source('hooks/useHashRoute.ts')
    expect(app).toContain('useHashRoute()')
    expect(app).not.toContain('subscribeTabChanges')
    expect(hook).toContain('setTabInLocation(activeTab, { replace: true })')
    expect(hook).toContain('subscribeTabChanges(setActiveTab)')
  })
})
