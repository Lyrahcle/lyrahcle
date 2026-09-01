import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import MissionSecurityOperationsDashboard from './MissionSecurityOperationsDashboard.jsx'

function renderDashboard(props = {}) {
  return render(
    <I18nProvider>
      <MissionSecurityOperationsDashboard
        satellites={[]}
        access={{ incidents: true, rf: true, agents: true, alerts: true }}
        {...props}
      />
    </I18nProvider>,
  )
}

describe('MissionSecurityOperationsDashboard', () => {
  it('summarizes cross-domain mission security posture and operator queue', () => {
    const now = Date.parse('2026-09-01T12:00:00Z')
    renderDashboard({
      now,
      live: true,
      backendStatus: 'online',
      satellites: [
        { id: 'sat-1', name: 'Alpha', updated_at: '2026-09-01T11:00:00Z', position: { latitude_deg: 1, longitude_deg: 2, altitude_km: 400 } },
        { id: 'sat-2', name: 'Bravo', updated_at: '2026-08-29T11:00:00Z', position: null },
      ],
      incidents: { items: [{ id: 'i1', reference: 'INC-1', title: 'Critical RF event', severity: 'critical', status: 'investigating', updated_at: '2026-09-01T11:50:00Z' }], total: 1 },
      anomalies: [{ id: 'a1', agent_id: 'sdr-01', metric: 'power_dbm', z_score: 6.2, detected_at: '2026-09-01T11:45:00Z' }],
      agentIdentities: [{ agent_id: 'sdr-01', status: 'active', last_seen_at: '2026-09-01T11:59:00Z' }],
      assetAlerts: { items: [{ id: 'v1', cve_id: 'CVE-2026-1000', asset_value: 'modem-1', severity: 'high', acknowledged: false, created_at: '2026-09-01T11:40:00Z' }], total: 1 },
    })

    expect(screen.getByRole('heading', { name: /mission security operations/i })).toBeInTheDocument()
    expect(screen.getAllByText('CRITICAL').length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText('Critical RF event').length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText('CVE-2026-1000').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('TLE stale')).toBeInTheDocument()
    expect(screen.getByText(/Unexpected RF anomaly · sdr-01/i)).toBeInTheDocument()
    expect(screen.getByText('1/1')).toBeInTheDocument()
    expect(screen.getByText('Live')).toBeInTheDocument()
  })

  it('keeps restricted domains hidden and reports partial access', () => {
    renderDashboard({
      satellites: [{ id: 'sat-1', name: 'Alpha', updated_at: new Date().toISOString(), position: { latitude_deg: 1, longitude_deg: 2, altitude_km: 400 } }],
      access: { incidents: false, rf: false, agents: false, alerts: false },
    })
    expect(screen.getByText(/0\/4 security data sources available/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /incidents/i })).not.toBeInTheDocument()
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(4)
  })

  it('navigates from quick actions and activity items', () => {
    const onNavigate = vi.fn()
    renderDashboard({
      onNavigate,
      incidents: { items: [{ id: 'i1', reference: 'INC-1', title: 'Investigate me', severity: 'high', status: 'new', updated_at: '2026-09-01T11:50:00Z' }], total: 1 },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }))
    expect(onNavigate).toHaveBeenCalledWith('incidents')
    const incidentSurfaces = screen.getAllByText('Investigate me')
    fireEvent.click(incidentSurfaces[incidentSurfaces.length - 1])
    expect(onNavigate).toHaveBeenCalledWith('incidents')
  })
})
