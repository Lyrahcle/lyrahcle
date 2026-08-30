import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CVESeverityChart, IncidentSeverityChart, RFPowerChart } from './TelemetryCharts.jsx'

describe('TelemetryCharts', () => {
  it('renders RF power telemetry', () => {
    render(<RFPowerChart readings={[
      { agent_id: 'sdr-1', power_dbm: -70, captured_at: '2026-08-30T10:00:00Z' },
      { agent_id: 'sdr-1', power_dbm: -62, captured_at: '2026-08-30T10:01:00Z' },
    ]} />)
    expect(screen.getByText('RF power trend')).toBeInTheDocument()
    expect(screen.getByText('-62.0 dBm')).toBeInTheDocument()
  })

  it('groups incidents by severity', () => {
    render(<IncidentSeverityChart incidents={[{ severity: 'high' }, { severity: 'high' }, { severity: 'low' }]} />)
    expect(screen.getByText('Current incident severity mix')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('groups visible CVEs by severity', () => {
    render(<CVESeverityChart cves={[{ severity: 'critical' }, { severity: 'medium' }]} />)
    expect(screen.getByText('Visible CVEs by severity')).toBeInTheDocument()
  })
})
