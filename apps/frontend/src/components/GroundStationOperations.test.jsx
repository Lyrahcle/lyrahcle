import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import GroundStationOperations from './GroundStationOperations.jsx'

vi.mock('./WorldMap.jsx', () => ({
  default: ({ stations = [], onStationSelect = () => {} }) => (
    <div>{stations.map((agent) => <button key={agent.agent_id} type="button" onClick={() => onStationSelect(agent)}>Map {agent.agent_id}</button>)}</div>
  ),
}))

function renderOperations(overrides = {}) {
  const now = new Date().toISOString()
  const props = {
    agents: [{
      agent_id: 'sdr-madrid',
      status: 'active',
      last_seen_at: now,
      station: { latitude_deg: 40.4168, longitude_deg: -3.7038, altitude_m: 667 },
      device: { hardware: 'RTL-SDR', compatibility: 'supported', capabilities: { integrity_status: 'verified' } },
    }],
    satellites: [{ id: 'sat-1', name: 'NOAA-19', norad_id: 33591, position: { latitude_deg: 40.4168, longitude_deg: -3.7038, altitude_km: 850 } }],
    readings: [{ agent_id: 'sdr-madrid', satellite_id: 'sat-1', frequency_hz: 137100000, power_dbm: -54, captured_at: now }],
    anomalies: [{ id: 'rf-1', agent_id: 'sdr-madrid', metric: 'power_dbm', z_score: 6, detected_at: now }],
    incidents: [{ id: 'inc-1', agent_id: 'sdr-madrid', title: 'Station signal anomaly', status: 'new', severity: 'high', created_at: now }],
    access: { orbital: true, rf: true, incidents: true },
    onNavigate: vi.fn(),
    ...overrides,
  }
  return { ...render(<I18nProvider><GroundStationOperations {...props} /></I18nProvider>), props }
}

describe('GroundStationOperations', () => {
  it('summarizes configured stations and current visibility', () => {
    renderOperations()
    expect(screen.getByRole('heading', { name: 'Ground Station Operations' })).toBeInTheDocument()
    expect(screen.getByText('Configured stations').parentElement.parentElement).toHaveTextContent('1')
    expect(screen.getByText('Satellites visible now').parentElement.parentElement).toHaveTextContent('1')
    expect(screen.getByText('Risk 18')).toBeInTheDocument()
  })

  it('opens a station profile with correlated RF and security evidence', () => {
    renderOperations()
    fireEvent.click(screen.getByRole('button', { name: 'Map sdr-madrid' }))
    expect(screen.getByRole('heading', { name: 'sdr-madrid' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Visible satellites now' })).toBeInTheDocument()
    expect(screen.getAllByText('NOAA-19').length).toBeGreaterThan(0)
    expect(screen.getByText('Station signal anomaly')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Recent RF captures · 24h' })).toBeInTheDocument()
    expect(screen.getByText('137.100 MHz')).toBeInTheDocument()
  })

  it('navigates to station configuration from the overview', () => {
    const { props } = renderOperations()
    fireEvent.click(screen.getByRole('button', { name: 'Manage station configuration' }))
    expect(props.onNavigate).toHaveBeenCalledWith('connectivity')
  })
})
