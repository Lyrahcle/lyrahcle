import { describe, expect, it } from 'vitest'
import { buildDashboardNavigation } from './navigation'

const t = (key: string) => key

describe('dashboard navigation builder', () => {
  it('keeps the core operator workspaces available', () => {
    const items = buildDashboardNavigation(t, { connectivity: false, administration: false, platform: false })
    expect(items.map((item) => item.id)).toEqual(['orbital', 'graph', 'assets', 'incidents', 'intel', 'signal', 'decoder', 'connectivity', 'settings'])
  })

  it('adds permission-driven workspaces only when available', () => {
    const items = buildDashboardNavigation(t, { connectivity: true, administration: true, platform: true })
    expect(items.map((item) => item.id)).toContain('stations')
    expect(items.map((item) => item.id)).toContain('admin')
    expect(items.map((item) => item.id)).toContain('platform')
  })
})
