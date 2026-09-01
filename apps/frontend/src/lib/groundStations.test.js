import { describe, expect, it } from 'vitest'
import { observedSatelliteIds, satelliteElevationDeg, stationCoordinates, visibleSatellites } from './groundStations.js'

describe('ground station operations helpers', () => {
  it('validates station coordinates', () => {
    expect(stationCoordinates({ station: { latitude_deg: 40.4, longitude_deg: -3.7, altitude_m: 650 } })).toEqual({ latitude: 40.4, longitude: -3.7, altitude: 650 })
    expect(stationCoordinates({ station: { latitude_deg: 100, longitude_deg: 0, altitude_m: 0 } })).toBeNull()
  })

  it('identifies an overhead satellite as visible', () => {
    const station = { latitude: 40, longitude: -3, altitude: 0 }
    const position = { latitude_deg: 40, longitude_deg: -3, altitude_km: 500 }
    expect(satelliteElevationDeg(station, position)).toBeGreaterThan(89)
    const visible = visibleSatellites({ station: { latitude_deg: 40, longitude_deg: -3, altitude_m: 0 } }, [{ id: 'sat-1', name: 'SAT-1', position }])
    expect(visible).toHaveLength(1)
  })

  it('deduplicates satellites observed by one station in the last day', () => {
    const now = Date.parse('2026-09-01T20:00:00Z')
    expect(observedSatelliteIds([
      { agent_id: 'sdr-1', satellite_id: 'sat-1', captured_at: '2026-09-01T19:00:00Z' },
      { agent_id: 'sdr-1', satellite_id: 'sat-1', captured_at: '2026-09-01T18:00:00Z' },
      { agent_id: 'sdr-2', satellite_id: 'sat-2', captured_at: '2026-09-01T19:00:00Z' },
    ], 'sdr-1', now)).toEqual(['sat-1'])
  })
})
