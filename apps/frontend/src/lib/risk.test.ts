import { describe, expect, it } from 'vitest'
import { calculateGroundStationRisk, calculateMissionRisk, calculateSatelliteRisk, riskLevel } from './risk'

describe('risk engine', () => {
  it('keeps score thresholds deterministic', () => {
    expect(riskLevel(0)).toBe('nominal')
    expect(riskLevel(35)).toBe('elevated')
    expect(riskLevel(70)).toBe('critical')
  })

  it('explains satellite risk with bounded factors', () => {
    const result = calculateSatelliteRisk({
      satellite: { tle_status: 'stale' },
      incidents: [{ status: 'open', severity: 'critical' }],
      anomalies: [{ detected_at: new Date().toISOString(), z_score: 6 }],
    })
    expect(result.score).toBe(48)
    expect(result.level).toBe('elevated')
    expect(result.factors.map((item) => item.key)).toEqual(['tleStale', 'criticalIncident', 'rfAnomaly'])
  })

  it('scores ground station risk from agent, incident and RF evidence', () => {
    const result = calculateGroundStationRisk({
      agent: { status: 'quarantined' },
      incidents: [{ status: 'new', severity: 'critical' }],
      anomalies: [{ detected_at: new Date().toISOString(), z_score: 6 }],
    })
    expect(result.score).toBe(58)
    expect(result.level).toBe('elevated')
    expect(result.state).toBe('attention')
  })

  it('aggregates mission risk across operational domains', () => {
    const result = calculateMissionRisk({
      satellites: [{ tle_status: 'aging' }],
      incidents: [{ status: 'open', severity: 'high' }],
      anomalies: [],
      agents: [],
      alerts: [],
    })
    expect(result.score).toBe(12)
    expect(result.domains.security).toBe(8)
    expect(result.domains.orbital).toBe(4)
  })
})
