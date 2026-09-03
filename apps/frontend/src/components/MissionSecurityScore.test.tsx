import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import MissionSecurityScore, { buildMissionSecurityScore } from './MissionSecurityScore'

const now = Date.parse('2026-09-02T06:00:00Z')

describe('MissionSecurityScore', () => {
  beforeEach(() => localStorage.clear())

  it('builds an explainable score from existing mission risk evidence', () => {
    const result = buildMissionSecurityScore({
      satellites: [{ tle_status: 'stale' }],
      incidents: [{ severity: 'critical', status: 'open' }],
      anomalies: [{ z_score: 6.2, detected_at: '2026-09-02T05:50:00Z' }],
      agents: [],
      alerts: [],
      access: { incidents: true, rf: true, agents: true, alerts: true },
      backendStatus: 'online',
      now,
    })
    expect(result.score).toBe(29)
    expect(result.domains.find((item) => item.domain === 'security')?.score).toBe(20)
    expect(result.domains.find((item) => item.domain === 'rf')?.score).toBe(5)
    expect(result.domains.find((item) => item.domain === 'orbital')?.score).toBe(4)
    expect(result.availableSources).toBe(6)
  })

  it('does not score evidence from domains the operator cannot access', () => {
    const result = buildMissionSecurityScore({
      satellites: [],
      incidents: [{ severity: 'critical', status: 'open' }],
      anomalies: [{ z_score: 7, detected_at: '2026-09-02T05:50:00Z' }],
      access: { incidents: false, rf: false, agents: false, alerts: false },
      backendStatus: 'online',
      now,
    })
    expect(result.score).toBe(0)
    expect(result.availableSources).toBe(2)
  })

  it('renders score, domains, coverage and evidence drivers', () => {
    render(
      <I18nProvider>
        <MissionSecurityScore
          satellites={[{ tle_status: 'stale' }]}
          incidents={[{ severity: 'high', status: 'open' }]}
          access={{ incidents: true, rf: true, agents: true, alerts: true }}
          backendStatus="online"
          now={now}
        />
      </I18nProvider>,
    )
    expect(screen.getByText('Mission Security Score')).toBeInTheDocument()
    expect(screen.getByText('6/6')).toBeInTheDocument()
    expect(screen.getByText('High-severity incidents')).toBeInTheDocument()
    expect(screen.getByText('Orbital')).toBeInTheDocument()
  })
})
