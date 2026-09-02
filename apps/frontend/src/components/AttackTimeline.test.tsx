import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import AttackTimeline, { buildAttackTimeline } from './AttackTimeline'

const readings = [{ id: 'r1', agent_id: 'gs-madrid', satellite_id: 'sat-1', frequency_hz: 437525000, power_dbm: -71.2, captured_at: '2026-09-02T06:01:00Z' }]
const anomalies = [{ id: 'a1', agent_id: 'gs-madrid', satellite_id: 'sat-1', metric: 'frequency_hz', z_score: 6.1, detected_at: '2026-09-02T06:02:00Z' }]
const incidents = [{ id: 'i1', satellite_id: 'sat-1', title: 'Unexpected carrier drift', severity: 'high', created_at: '2026-09-02T06:03:00Z', status: 'open' }]

function renderTimeline() {
  return render(<I18nProvider><AttackTimeline readings={readings} anomalies={anomalies} incidents={incidents} /></I18nProvider>)
}

describe('AttackTimeline', () => {
  it('merges real RF captures, anomalies and incidents in descending time order', () => {
    const events = buildAttackTimeline(readings, anomalies, incidents)
    expect(events.map((event) => event.kind)).toEqual(['incident', 'anomaly', 'capture'])
    expect(events[1].detail).toContain('6.1σ')
  })

  it('renders multimodal evidence without fabricating missing values', () => {
    renderTimeline()
    expect(screen.getByText('437.525000 MHz · -71.2 dBm')).toBeInTheDocument()
    expect(screen.getByText('frequency_hz · 6.1σ')).toBeInTheDocument()
    expect(screen.getByText('Unexpected carrier drift')).toBeInTheDocument()
  })

  it('filters the timeline by evidence type', async () => {
    const user = userEvent.setup()
    renderTimeline()
    await user.selectOptions(screen.getByLabelText('Evidence type'), 'anomaly')
    expect(screen.getByText('frequency_hz · 6.1σ')).toBeInTheDocument()
    expect(screen.queryByText('437.525000 MHz · -71.2 dBm')).not.toBeInTheDocument()
    expect(screen.queryByText('Unexpected carrier drift')).not.toBeInTheDocument()
  })
})
