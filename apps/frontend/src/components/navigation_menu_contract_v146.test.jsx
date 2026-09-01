import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

function source(relativePath) {
  return readFileSync(resolve(process.cwd(), 'src', relativePath), 'utf8')
}

describe('navigation shell contract v1.46', () => {
  it('uses a persistent sidebar instead of horizontal primary tabs', () => {
    const dashboard = source('components/Dashboard.jsx')
    const header = source('components/Header.jsx')
    expect(dashboard).not.toContain("import Tabs from './Tabs.jsx'")
    expect(dashboard).toContain('<NavigationMenu')
    expect(dashboard).toContain("lg:pl-20")
    expect(dashboard).toContain("lg:pl-64")
    expect(header).toContain('onOpenNavigation')
  })

  it('keeps orbital as the home route and settings in page navigation', () => {
    const app = source('App.jsx')
    const dashboard = source('components/Dashboard.jsx')
    const routeTabs = source('lib/routeTabs.ts')
    expect(routeTabs).toContain("export const DEFAULT_TAB = 'orbital'")
    expect(app).toContain('getTabFromLocation(DEFAULT_TAB)')
    expect(dashboard).toContain("onHome={() => onChangeTab('orbital')}")
    expect(dashboard).toContain("{ id: 'settings', label: t('tabs.settings'), section: 'preferences' }")
  })

  it('keeps the active section deep-linkable via the URL hash', () => {
    const app = source('App.jsx')
    expect(app).toContain('setTabInLocation(activeTab, { replace: true })')
    expect(app).toContain('subscribeTabChanges(setActiveTab)')
  })
})
