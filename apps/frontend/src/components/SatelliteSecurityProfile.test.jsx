import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SatelliteSecurityProfile from './SatelliteSecurityProfile.jsx'
import { I18nProvider } from '../lib/I18nContext'
import * as api from '../lib/api'

vi.mock('../lib/api', () => ({ fetchOrbitalEvents: vi.fn(), fetchIncidents: vi.fn() }))

const satellite = {
  id: 'sat-1', norad_id: 25544, name: 'ISS', tle_status: 'aging', orbital_confidence: 'high',
  updated_at: '2026-09-01T17:00:00Z', position: { latitude_deg: 40.2, longitude_deg: -3.7, altitude_km: 418.7 },
}

function renderProfile(props = {}) {
  return render(<I18nProvider><SatelliteSecurityProfile satellite={satellite} now={Date.parse('2026-09-01T18:00:00Z')} {...props} /></I18nProvider>)
}

describe('SatelliteSecurityProfile', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    api.fetchIncidents.mockResolvedValue({ items: [{ id: 'inc-1', satellite_id: 'sat-1', title: 'Investigate uplink', severity: 'critical', status: 'new', created_at: '2026-09-01T17:10:00Z' }], total: 1, limit: 50, offset: 0 })
    api.fetchOrbitalEvents.mockResolvedValue([{ id: 'evt-1', satellite_id: 'sat-1', event_type: 'maneuver', event_time: '2026-09-01T17:30:00Z', source: 'operator', confidence: 0.9, severity: 'high' }])
  })

  it('correlates satellite incidents, RF anomalies, orbital events and exact exposure matches', async () => {
    renderProfile({
      access: { incidents: true, rf: true, agents: true, alerts: true },
      incidents: { items: [{ id: 'inc-1', satellite_id: 'sat-1', title: 'Investigate uplink', severity: 'critical', status: 'new', created_at: '2026-09-01T17:10:00Z' }, { id: 'inc-2', satellite_id: 'other', title: 'Other', severity: 'critical', status: 'new' }] },
      anomalies: [{ id: 'rf-1', satellite_id: 'sat-1', agent_id: 'sdr-1', metric: 'frequency_hz', z_score: 6.2, detected_at: '2026-09-01T17:20:00Z' }],
      agentIdentities: [{ agent_id: 'sdr-1', status: 'active', last_seen_at: '2026-09-01T17:59:00Z' }],
      assetAlerts: { items: [{ id: 'a-1', asset_value: 'ISS', cve_id: 'CVE-2026-1234', matched_keyword: 'radio', severity: 'high', acknowledged: false, created_at: '2026-09-01T17:40:00Z' }, { id: 'a-2', asset_value: 'OTHER', cve_id: 'CVE-2026-9999', severity: 'critical', acknowledged: false }] },
    })

    expect(screen.getByRole('heading', { name: 'ISS' })).toBeInTheDocument()
    expect(screen.getByText('Investigate uplink')).toBeInTheDocument()
    expect(screen.getByText('CVE-2026-1234')).toBeInTheDocument()
    expect(screen.queryByText('CVE-2026-9999')).not.toBeInTheDocument()
    expect(screen.getByText('sdr-1')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByText('maneuver')).toBeInTheDocument())
    expect(api.fetchOrbitalEvents).toHaveBeenCalledWith('sat-1', 168, 50)
  })

  it('supports back and cross-domain navigation', async () => {
    const onBack = vi.fn()
    const onNavigate = vi.fn()
    renderProfile({ access: { incidents: true, rf: true }, onBack, onNavigate })
    await waitFor(() => expect(api.fetchOrbitalEvents).toHaveBeenCalledWith('sat-1', 168, 50))
    await waitFor(() => expect(api.fetchIncidents).toHaveBeenCalledWith(50, 0, { satelliteId: 'sat-1' }))
    await waitFor(() => expect(screen.getByText('maneuver')).toBeInTheDocument())
    fireEvent.click(screen.getByRole('button', { name: /back to orbital operations/i }))
    fireEvent.click(screen.getByRole('button', { name: /open incidents/i }))
    fireEvent.click(screen.getByRole('button', { name: /open rf detection/i }))
    expect(onBack).toHaveBeenCalledOnce()
    expect(onNavigate).toHaveBeenNthCalledWith(1, 'incidents')
    expect(onNavigate).toHaveBeenNthCalledWith(2, 'signal')
  })
})
