import { describe, expect, it } from 'vitest'
import { buildInvestigationChain } from './IncidentInvestigationChain'

describe('buildInvestigationChain', () => {
  it('preserves recorded evidence references and chronological history', () => {
    const chain = buildInvestigationChain({
      incident: { id: 'inc-1', reference: 'INC-001' },
      evidence: [
        { id: 'e1', evidence_type: 'rf_anomaly', evidence_ref: 'a-1', label: 'RF anomaly' },
        { id: 'e2', evidence_type: 'orbital_event', evidence_ref: 'o-1', label: 'maneuver candidate' },
      ],
      events: [
        { id: 'later', event_type: 'note', created_at: '2026-09-02T11:00:00Z' },
        { id: 'earlier', event_type: 'created', created_at: '2026-09-02T10:00:00Z' },
      ],
    })
    expect(chain.reference).toBe('INC-001')
    expect(chain.nodes.map((node) => node.reference)).toEqual(['a-1', 'o-1'])
    expect(chain.history.map((event) => event.id)).toEqual(['earlier', 'later'])
  })

  it('does not infer missing causal links', () => {
    const chain = buildInvestigationChain({ incident: { id: 'inc-1' }, evidence: [], events: [] })
    expect(chain.nodes).toEqual([])
    expect(chain.history).toEqual([])
    expect(chain.domains).toEqual([])
  })
})
