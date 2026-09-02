import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import RFDopplerPanel from './RFDopplerPanel'

const comparison = {
  anomaly_id: 'a1', reading_id: 'r1', agent_id: 'gs-madrid', satellite_id: 'sat-1', satellite_name: 'SAT-1',
  observed_frequency_hz: 437_504_000, baseline_frequency_hz: 437_500_000, expected_doppler_hz: 1_000,
  expected_frequency_hz: 437_501_000, residual_hz: 3_000, tolerance_hz: 2_500,
  range_rate_mps: -685, elevation_deg: 42.5, visible: true, tle_status: 'fresh' as const,
  detected_at: '2026-09-02T12:00:00Z',
}

describe('RFDopplerPanel', () => {
  it('renders expected and observed frequency without synthesizing missing data', () => {
    render(<I18nProvider><RFDopplerPanel comparisons={[comparison]} /></I18nProvider>)
    expect(screen.getByText('SAT-1')).toBeInTheDocument()
    expect(screen.getByText('437.504000 MHz')).toBeInTheDocument()
    expect(screen.getByText('437.501000 MHz')).toBeInTheDocument()
    expect(screen.getByText('Outside tolerance')).toBeInTheDocument()
  })

  it('renders an explicit empty state when orbital Doppler context is unavailable', () => {
    render(<I18nProvider><RFDopplerPanel comparisons={[]} /></I18nProvider>)
    expect(screen.getByText('No Doppler comparisons available.')).toBeInTheDocument()
  })
})
