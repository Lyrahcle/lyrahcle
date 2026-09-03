import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import RFAttackTriage, { classifyRFAttackCandidates } from './RFAttackTriage'

const anomalies = [
  { id: 'a1', agent_id: 'gs-madrid', metric: 'power_dbm', z_score: 6.4, detected_at: '2026-09-02T04:00:00Z' },
  { id: 'a2', agent_id: 'gs-madrid', metric: 'frequency_hz', z_score: -5.6, detected_at: '2026-09-02T04:01:00Z' },
  { id: 'a3', agent_id: 'gs-paris', metric: 'power_dbm', z_score: -7.1, detected_at: '2026-09-02T04:02:00Z' },
  { id: 'a4', agent_id: 'gs-paris', metric: 'frequency_hz_trend', z_score: 4.2, detected_at: '2026-09-02T04:03:00Z' },
]

describe('RFAttackTriage', () => {
  beforeEach(() => localStorage.clear())

  it('classifies only evidence-supported jamming and spoofing candidates', () => {
    const result = classifyRFAttackCandidates(anomalies)
    expect(result.map((item) => item.kind)).toEqual(['jamming', 'spoofing', 'spoofing'])
    expect(result.some((item) => item.id.includes('a3'))).toBe(false)
  })

  it('renders candidate evidence and does not fabricate replay detection', () => {
    render(<I18nProvider><RFAttackTriage anomalies={anomalies} /></I18nProvider>)
    expect(screen.getByText('RF Attack Triage')).toBeInTheDocument()
    expect(screen.getByText('Jamming candidate')).toBeInTheDocument()
    expect(screen.getAllByText('Spoofing candidate')).toHaveLength(2)
    expect(screen.getByText('Replay detection needs frame or sequence evidence that is not present in RF anomaly records.')).toBeInTheDocument()
    expect(screen.getByText('gs-madrid · power_dbm · +6.4σ')).toBeInTheDocument()
  })

  it('shows an evidence-empty state when no anomaly crosses the triage threshold', () => {
    render(<I18nProvider><RFAttackTriage anomalies={[{ id: 'a1', metric: 'power_dbm', z_score: 2.1 }]} /></I18nProvider>)
    expect(screen.getByText('No RF anomaly currently meets the attack-triage threshold.')).toBeInTheDocument()
  })
})
