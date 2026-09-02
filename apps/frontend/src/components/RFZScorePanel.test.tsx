import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import RFZScorePanel from './RFZScorePanel'
import { I18nProvider } from '../lib/I18nContext'

const anomaly = { id: 'a1', reading_id: 'r1', agent_id: 'sdr-01', metric: 'power_dbm', value: -20, baseline_mean: -60, baseline_stddev: 5, z_score: 8, detected_at: '2026-09-02T12:00:00Z' }

describe('RFZScorePanel', () => {
  it('renders the strongest z-score and operational counts', () => {
    render(<I18nProvider><RFZScorePanel anomalies={[anomaly]} /></I18nProvider>)
    expect(screen.getAllByText('8.0σ').length).toBeGreaterThan(0)
    expect(screen.getByText('sdr-01')).toBeInTheDocument()
  })

  it('renders an explicit empty state without fabricating samples', () => {
    render(<I18nProvider><RFZScorePanel anomalies={[]} /></I18nProvider>)
    expect(screen.getByText(/No z-score anomalies available/i)).toBeInTheDocument()
  })
})
