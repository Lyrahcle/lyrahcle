import { describe, expect, it } from 'vitest'
import { buildIncidentEvidencePackage } from './IncidentEvidenceCase'

describe('buildIncidentEvidencePackage', () => {
  it('keeps candidate evidence non-confirmatory and reproducible', () => {
    const pkg = buildIncidentEvidencePackage({
      satellite: { id: 'sat-1', name: 'SAT-1' },
      anomalies: [{ id: 'a-1', metric: 'power_dbm', z_score: 5.2 }],
      orbitalEvents: [{ id: 'o-1', event_type: 'maneuver' }],
      tmtcAnalysis: { replay_candidates: 1, issues: [{ kind: 'replay_candidate' }] },
    })
    expect(pkg.title).toContain('SAT-1')
    expect(pkg.description).toMatch(/does not assert causality/i)
    expect(pkg.evidence.map((item) => item.type)).toEqual(expect.arrayContaining(['rf_anomaly', 'orbital_event', 'tmtc_session']))
  })

  it('does not fabricate evidence references when records have no ids', () => {
    const pkg = buildIncidentEvidencePackage({
      satellite: { id: 'sat-1', name: 'SAT-1' },
      anomalies: [{ metric: 'power_dbm', z_score: 5.2 }],
      orbitalEvents: [{ event_type: 'maneuver' }],
    })
    expect(pkg.evidence).toHaveLength(0)
  })
})
