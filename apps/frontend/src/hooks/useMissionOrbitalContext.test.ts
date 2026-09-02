import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchOrbitalEvents, fetchSatellitePasses } from '../lib/api'
import type { OrbitalEvent, SatellitePass, SatellitePassPredictionResponse } from '../lib/apiTypes'
import { useMissionOrbitalContext } from './useMissionOrbitalContext'

vi.mock('../lib/api', () => ({
  fetchSatellitePasses: vi.fn(),
  fetchOrbitalEvents: vi.fn(),
}))

const coordinates = { latitude: 40.4168, longitude: -3.7038, altitude: 667 }

function orbitalEvent(id: string): OrbitalEvent {
  return {
    id,
    satellite_id: 'sat-1',
    event_type: 'maneuver',
    event_time: '2026-09-02T09:00:00Z',
    source: 'operator',
    confidence: 0.8,
    severity: 'medium',
    metadata: {},
    created_at: '2026-09-02T09:01:00Z',
  }
}

function pass(id = 'pass-1'): SatellitePass & { id: string } {
  return {
    id,
    aos: '2026-09-02T10:00:00Z',
    los: '2026-09-02T10:10:00Z',
    peak_at: '2026-09-02T10:05:00Z',
    max_elevation_deg: 42,
    duration_seconds: 600,
  }
}

function passResponse(passes: SatellitePass[]): SatellitePassPredictionResponse {
  return {
    satellite_id: 'sat-1',
    norad_id: 25544,
    satellite_name: 'ISS',
    tle_status: 'fresh',
    tle_age_seconds: 60,
    orbital_confidence: 'high',
    station: { latitude_deg: 40.4168, longitude_deg: -3.7038, altitude_m: 667 },
    from: '2026-09-02T09:00:00Z',
    until: '2026-09-03T09:00:00Z',
    min_elevation_deg: 0,
    passes,
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useMissionOrbitalContext', () => {
  beforeEach(() => {
    vi.mocked(fetchOrbitalEvents).mockReset()
    vi.mocked(fetchSatellitePasses).mockReset()
  })

  it('loads orbital events and pass predictions for the selected satellite', async () => {
    vi.mocked(fetchOrbitalEvents).mockResolvedValue([orbitalEvent('event-1')])
    vi.mocked(fetchSatellitePasses).mockResolvedValue(passResponse([pass()]))

    const { result } = renderHook(() => useMissionOrbitalContext('sat-1', coordinates, true))

    await waitFor(() => expect(result.current.orbitalEvents).toEqual([orbitalEvent('event-1')]))
    await waitFor(() => expect(result.current.passes).toEqual([pass()]))
    expect(fetchOrbitalEvents).toHaveBeenCalledWith('sat-1', 168, 50)
    expect(fetchSatellitePasses).toHaveBeenCalledWith('sat-1', expect.objectContaining({ latitudeDeg: coordinates.latitude, longitudeDeg: coordinates.longitude }))
  })

  it('degrades failed requests to empty evidence instead of leaking rejected state', async () => {
    vi.mocked(fetchOrbitalEvents).mockRejectedValue(new Error('orbital unavailable'))
    vi.mocked(fetchSatellitePasses).mockRejectedValue(new Error('passes unavailable'))

    const { result } = renderHook(() => useMissionOrbitalContext('sat-1', coordinates, true))

    await waitFor(() => expect(result.current.orbitalEvents).toEqual([]))
    await waitFor(() => expect(result.current.passes).toEqual([]))
  })

  it('does not request pass predictions without trustworthy station coordinates', async () => {
    vi.mocked(fetchOrbitalEvents).mockResolvedValue([])

    const { result } = renderHook(() => useMissionOrbitalContext('sat-1', null, true))

    await waitFor(() => expect(result.current.orbitalEvents).toEqual([]))
    expect(result.current.passes).toEqual([])
    expect(fetchSatellitePasses).not.toHaveBeenCalled()
  })

  it('ignores stale responses when the selected satellite changes', async () => {
    const oldEvents = deferred<OrbitalEvent[]>()
    const newEvents = deferred<OrbitalEvent[]>()
    const oldPasses = deferred<SatellitePassPredictionResponse>()
    const newPasses = deferred<SatellitePassPredictionResponse>()
    vi.mocked(fetchOrbitalEvents).mockImplementation((satelliteId) => satelliteId === 'sat-1' ? oldEvents.promise : newEvents.promise)
    vi.mocked(fetchSatellitePasses).mockImplementation((satelliteId) => satelliteId === 'sat-1' ? oldPasses.promise : newPasses.promise)

    const { result, rerender } = renderHook(
      ({ satelliteId }) => useMissionOrbitalContext(satelliteId, coordinates, true),
      { initialProps: { satelliteId: 'sat-1' } },
    )

    rerender({ satelliteId: 'sat-2' })
    await act(async () => {
      newEvents.resolve([orbitalEvent('new-event')])
      newPasses.resolve(passResponse([pass('new-pass')]))
    })
    await waitFor(() => expect(result.current.orbitalEvents).toEqual([orbitalEvent('new-event')]))
    await waitFor(() => expect(result.current.passes).toEqual([pass('new-pass')]))

    await act(async () => {
      oldEvents.resolve([orbitalEvent('stale-event')])
      oldPasses.resolve(passResponse([pass('stale-pass')]))
    })
    expect(result.current.orbitalEvents).toEqual([orbitalEvent('new-event')])
    expect(result.current.passes).toEqual([pass('new-pass')])
  })

  it('ignores in-flight responses after unmount', async () => {
    const events = deferred<OrbitalEvent[]>()
    const passes = deferred<SatellitePassPredictionResponse>()
    vi.mocked(fetchOrbitalEvents).mockReturnValue(events.promise)
    vi.mocked(fetchSatellitePasses).mockReturnValue(passes.promise)

    const { unmount } = renderHook(() => useMissionOrbitalContext('sat-1', coordinates, true))
    unmount()

    await act(async () => {
      events.resolve([orbitalEvent('late-event')])
      passes.resolve(passResponse([pass('late-pass')]))
    })
  })
})
