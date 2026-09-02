import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useDashboardSubscriptions } from './useDashboardSubscriptions'

const { subscribeSatellites, subscribeIncidents } = vi.hoisted(() => ({
  subscribeSatellites: vi.fn(),
  subscribeIncidents: vi.fn(),
}))

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api')
  return { ...actual, subscribeSatellites, subscribeIncidents }
})

function options(overrides = {}) {
  return {
    activeTab: 'assets' as const,
    onSessionExpired: vi.fn(),
    setSatellites: vi.fn(),
    setSatellitesLive: vi.fn(),
    setSelectedSatelliteId: vi.fn(),
    setIncidents: vi.fn(),
    incidentsOffsetRef: { current: 0 },
    incidentFiltersRef: { current: { status: '', severity: '', agentId: '', satelliteId: '' } },
    ...overrides,
  }
}

describe('useDashboardSubscriptions', () => {
  beforeEach(() => {
    subscribeSatellites.mockReset().mockReturnValue(vi.fn())
    subscribeIncidents.mockReset().mockReturnValue(vi.fn())
  })

  it('subscribes to satellite live data only in the orbital workspace', () => {
    const props = options({ activeTab: 'orbital' as const })
    renderHook(() => useDashboardSubscriptions(props))
    expect(subscribeSatellites).toHaveBeenCalledTimes(1)
    expect(subscribeIncidents).not.toHaveBeenCalled()
  })

  it('resets orbital-only UI state when leaving the orbital workspace', () => {
    const props = options()
    renderHook(() => useDashboardSubscriptions(props))
    expect(props.setSatellitesLive).toHaveBeenCalledWith(false)
    expect(props.setSelectedSatelliteId).toHaveBeenCalledWith(null)
  })

  it('subscribes to incident live data only in the incidents workspace', () => {
    const props = options({ activeTab: 'incidents' as const })
    renderHook(() => useDashboardSubscriptions(props))
    expect(subscribeIncidents).toHaveBeenCalledTimes(1)
    expect(subscribeSatellites).not.toHaveBeenCalled()
  })
})
