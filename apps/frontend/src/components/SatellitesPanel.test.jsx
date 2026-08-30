import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SatellitesPanel from './SatellitesPanel.jsx'
import { I18nProvider } from '../lib/I18nContext.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({ updateSatellite: vi.fn() }))

const satellite = {
  id: 'sat-1',
  norad_id: 25544,
  name: 'ISS',
  tle_line1: '1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995',
  tle_line2: '2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754',
  tle_status: 'fresh',
  orbital_confidence: 'high',
  updated_at: '2026-08-30T18:00:00Z',
  position: { latitude_deg: 40.2, longitude_deg: -3.7, altitude_km: 418.7 },
}

function renderPanel(props = {}) {
  return render(<I18nProvider><SatellitesPanel satellites={[satellite]} {...props} /></I18nProvider>)
}

describe('SatellitesPanel management', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('does not expose edit controls without orbital.manage', () => {
    renderPanel({ canManage: false })
    expect(screen.getByText('ISS')).toBeInTheDocument()
    expect(screen.getByText('fresh')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit ISS/i })).not.toBeInTheDocument()
  })

  it('edits name, NORAD and TLE from the UI for orbital.manage', async () => {
    api.updateSatellite.mockResolvedValue({ ...satellite, name: 'ISS (ZARYA)' })
    const onChanged = vi.fn().mockResolvedValue()
    renderPanel({ canManage: true, onChanged })

    fireEvent.click(screen.getByRole('button', { name: /edit ISS/i }))
    fireEvent.change(screen.getByLabelText(/^Name$/i), { target: { value: ' ISS (ZARYA) ' } })
    fireEvent.change(screen.getByLabelText(/TLE line 1/i), { target: { value: ` ${satellite.tle_line1} ` } })
    fireEvent.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => expect(api.updateSatellite).toHaveBeenCalledWith('sat-1', {
      noradId: 25544,
      name: 'ISS (ZARYA)',
      tleLine1: satellite.tle_line1,
      tleLine2: satellite.tle_line2,
    }))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('shows update failures without closing the editor', async () => {
    api.updateSatellite.mockRejectedValue(new Error('NORAD ID is already registered'))
    renderPanel({ canManage: true })
    fireEvent.click(screen.getByRole('button', { name: /edit ISS/i }))
    fireEvent.click(screen.getByRole('button', { name: /save changes/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('NORAD ID is already registered')
    expect(screen.getByRole('button', { name: /save changes/i })).toBeInTheDocument()
  })
})
