import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { AnomalyTimelineChart, CVESeverityChart, CVETimelineChart, IncidentSeverityChart, IncidentTimelineChart, OrbitalAltitudeChart, RFPowerChart, RFSpectrumChart, RFStationMapChart } from './TelemetryCharts.jsx'

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

  it('renders RF spectrum frequencies', () => {
    render(<RFSpectrumChart readings={[
      { frequency_hz: 437500000, power_dbm: -65 },
      { frequency_hz: 437600000, power_dbm: -55 },
    ]} />)
    expect(screen.getByText('RF spectrum snapshot')).toBeInTheDocument()
  })

  it('renders anomaly timeline', () => {
    render(<AnomalyTimelineChart anomalies={[
      { agent_id: 'sdr-1', z_score: 4.5, detected_at: '2026-08-30T10:00:00Z' },
    ]} />)
    expect(screen.getByText('Anomaly intensity trend')).toBeInTheDocument()
  })

  it('renders incident and CVE timelines', () => {
    const { rerender } = render(<IncidentTimelineChart incidents={[{ created_at: '2026-08-30T10:00:00Z' }]} />)
    expect(screen.getByText('Incidents created over time')).toBeInTheDocument()
    rerender(<CVETimelineChart cves={[{ published: '2026-08-29T10:00:00Z' }]} />)
    expect(screen.getByText('CVE publication timeline')).toBeInTheDocument()
  })

  it('renders configured RF stations', () => {
    render(<RFStationMapChart
      agents={[{ agent_id: 'sdr-1', station: { latitude_deg: 40.4, longitude_deg: -3.7 } }]}
      readings={[{ agent_id: 'sdr-1', power_dbm: -61, captured_at: '2026-08-30T10:00:00Z' }]}
    />)
    expect(screen.getByText('RF station coverage')).toBeInTheDocument()
    expect(screen.getByText('1 configured station')).toBeInTheDocument()
  })

  it('renders orbital altitude profile', () => {
    render(<OrbitalAltitudeChart satellites={[
      { name: 'SAT-1', position: { altitude_km: 550 } },
      { name: 'SAT-2', position: { altitude_km: 420 } },
    ]} />)
    expect(screen.getByText('Orbital altitude profile')).toBeInTheDocument()
    expect(screen.getByText('550 km')).toBeInTheDocument()
  })
})
