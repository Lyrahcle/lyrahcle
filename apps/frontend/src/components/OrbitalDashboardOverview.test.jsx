import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import OrbitalDashboardOverview from './OrbitalDashboardOverview.tsx'

function renderPanel(props = {}) {
  return render(<I18nProvider><OrbitalDashboardOverview {...props} /></I18nProvider>)
}

describe('OrbitalDashboardOverview', () => {
  it('summarizes live constellation posture and stale TLE data', () => {
    const now = Date.parse('2026-08-31T12:00:00Z')
    renderPanel({
      now,
      live: true,
      satellites: [
        { id: 'a', name: 'Alpha', norad_id: 1, updated_at: '2026-08-31T11:00:00Z', position: { latitude_deg: 1, longitude_deg: 2, altitude_km: 400 } },
        { id: 'b', name: 'Bravo', norad_id: 2, updated_at: '2026-08-29T10:00:00Z', position: null },
      ],
    })
    expect(screen.getByRole('heading', { name: /orbital dashboard/i })).toBeInTheDocument()
    expect(screen.getAllByText('1/2')).toHaveLength(2)
    expect(screen.getByText('TLE stale')).toBeInTheDocument()
    expect(screen.getByText('Live')).toBeInTheDocument()
  })

  it('renders a healthy empty attention queue for fresh positioned satellites', () => {
    const now = Date.parse('2026-08-31T12:00:00Z')
    renderPanel({ now, satellites: [{ id: 'a', name: 'Alpha', updated_at: '2026-08-31T11:00:00Z', position: { latitude_deg: 1, longitude_deg: 2, altitude_km: 400 } }] })
    expect(screen.getByText(/no orbital data issues require attention/i)).toBeInTheDocument()
  })
})
