import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import MissionControl from './MissionControl'

vi.mock('../lib/api', () => ({
  fetchSatellitePasses: vi.fn().mockResolvedValue([{ aos_at: '2026-09-02T10:00:00Z', los_at: '2026-09-02T10:10:00Z' }]),
  fetchOrbitalEvents: vi.fn().mockResolvedValue([]),
}))

function renderControl(props: Record<string, unknown> = {}) {
  return render(<I18nProvider><MissionControl
    satellites={[{ id: 'sat-1', name: 'SAT-1', norad_id: 12345, updated_at: '2026-09-02T09:00:00Z', position: { latitude_deg: 40, longitude_deg: -3, altitude_km: 500 } }]}
    readings={[]}
    anomalies={[]}
    incidents={{ items: [] }}
    agents={[]}
    assetAlerts={{ items: [] }}
    access={{ orbital: true, rf: true, incidents: true, agents: true, alerts: true }}
    now={Date.parse('2026-09-02T09:30:00Z')}
    {...props}
  /></I18nProvider>)
}

describe('MissionControl', () => {
  it('renders a satellite-centric mission workspace without inventing missing station data', () => {
    renderControl()
    expect(screen.getByRole('heading', { name: /mission control/i })).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /satellite/i })).toHaveValue('sat-1')
    expect(screen.getByText('12345')).toBeInTheDocument()
    expect(screen.getAllByText('—').length).toBeGreaterThan(0)
  })

  it('correlates RF evidence and the related ground station', async () => {
    renderControl({
      readings: [{ id: 'r1', satellite_id: 'sat-1', agent_id: 'gs-madrid', frequency_hz: 437525000, power_dbm: -71.2, captured_at: '2026-09-02T09:29:00Z' }],
      anomalies: [{ id: 'a1', satellite_id: 'sat-1', agent_id: 'gs-madrid', z_score: 6.1, detected_at: '2026-09-02T09:28:00Z' }],
      agents: [{ agent_id: 'gs-madrid', station: { latitude_deg: 40.4, longitude_deg: -3.7, altitude_m: 650 } }],
    })
    expect(screen.getAllByText('gs-madrid').length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText('437.525000 MHz')).toBeInTheDocument()
    expect(screen.getByText('-71.2 dBm')).toBeInTheDocument()
    expect(screen.getByText('6.1σ')).toBeInTheDocument()
    await waitFor(() => expect(screen.queryByText('…')).not.toBeInTheDocument())
  })

  it('surfaces candidate orbital evidence without presenting it as a confirmed attack', async () => {
    const api = await import('../lib/api')
    vi.mocked(api.fetchOrbitalEvents).mockResolvedValueOnce([{ id: 'oe-1', event_type: 'maneuver', event_time: '2026-09-02T09:27:00Z', source: 'automatic-screening', confidence: 0.82, severity: 'medium', metadata: { candidate_only: true } }])
    renderControl()
    expect((await screen.findAllByText(/maneuver candidate/i)).length).toBeGreaterThan(0)
    expect(screen.getByText(/automatic-screening · 82%/i)).toBeInTheDocument()
  })

  it('opens existing operational workspaces from Mission Control', () => {
    const onNavigate = vi.fn()
    renderControl({ onNavigate })
    fireEvent.click(screen.getByRole('button', { name: /open rf workspace/i }))
    expect(onNavigate).toHaveBeenCalledWith('signal')
  })
})
