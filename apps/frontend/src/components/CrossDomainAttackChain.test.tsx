import { describe, expect, it } from 'vitest'
import { buildCrossDomainAttackChain } from './CrossDomainAttackChain'

describe('buildCrossDomainAttackChain', () => {
  it('keeps unavailable domains explicit when evidence is missing', () => {
    const stages = buildCrossDomainAttackChain({})
    expect(stages).toHaveLength(4)
    expect(stages.every((stage) => stage.state === 'unavailable')).toBe(true)
  })

  it('surfaces TMTC replay and orbital evidence as candidates, not confirmed attacks', () => {
    const stages = buildCrossDomainAttackChain({
      tmtcAnalysis: { issues: [{ kind: 'replay_candidate', replay_candidate: true }], replay_candidates: 1 },
      orbitalEvents: [{ event_type: 'maneuver' }],
    })
    expect(stages.find((stage) => stage.id === 'tmtc')?.state).toBe('candidate')
    expect(stages.find((stage) => stage.id === 'orbital')?.state).toBe('candidate')
    expect(JSON.stringify(stages).toLowerCase()).not.toContain('confirmed attack')
  })

  it('surfaces active incident evidence independently of attack inference', () => {
    const stages = buildCrossDomainAttackChain({ incidents: [{ status: 'open', title: 'Operator review' }, { status: 'resolved', title: 'Closed' }] })
    const incident = stages.find((stage) => stage.id === 'incident')
    expect(incident?.state).toBe('observed')
    expect(incident?.evidence).toEqual(['Operator review'])
  })
})
