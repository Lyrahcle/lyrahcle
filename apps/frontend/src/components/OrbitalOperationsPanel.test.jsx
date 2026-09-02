import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import { clearEffectivePermissions, setEffectivePermissions } from '../lib/permissions'
import OrbitalOperationsPanel from './OrbitalOperationsPanel.tsx'
import * as api from '../lib/api'

vi.mock('../lib/api', () => ({
  createOrbitalEvent: vi.fn(),
  fetchOrbitalEvents: vi.fn(),
  fetchSatellitePasses: vi.fn(),
}))

const satellites = [{ id: 'sat-1', name: 'ISS', norad_id: 25544 }]

function renderPanel() {
  return render(<I18nProvider><OrbitalOperationsPanel satellites={satellites} /></I18nProvider>)
}

describe('OrbitalOperationsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    clearEffectivePermissions()
    api.fetchOrbitalEvents.mockResolvedValue([])
  })

  it('runs pass prediction from the UI for orbital.read', async () => {
    setEffectivePermissions(['orbital.read'])
    api.fetchSatellitePasses.mockResolvedValue({
      tle_status: 'fresh',
      orbital_confidence: 'high',
      passes: [{
        aos: '2026-08-30T18:00:00Z',
        los: '2026-08-30T18:10:00Z',
        peak_at: '2026-08-30T18:05:00Z',
        max_elevation_deg: 47.2,
        duration_seconds: 600,
      }],
    })
    renderPanel()

    fireEvent.change(screen.getByLabelText(/station latitude/i), { target: { value: '40.4168' } })
    fireEvent.change(screen.getByLabelText(/station longitude/i), { target: { value: '-3.7038' } })
    fireEvent.change(screen.getByLabelText(/altitude/i), { target: { value: '667' } })
    fireEvent.click(screen.getByRole('button', { name: /^predict passes$/i }))

    await waitFor(() => expect(api.fetchSatellitePasses).toHaveBeenCalledWith('sat-1', {
      latitudeDeg: 40.4168,
      longitudeDeg: -3.7038,
      altitudeM: 667,
      hours: 24,
      minElevationDeg: 10,
      limit: 10,
    }))
    expect(await screen.findByText('47.2°')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /add orbital event/i })).not.toBeInTheDocument()
  })

  it('lets orbital.manage create candidate orbital events and refreshes the list', async () => {
    setEffectivePermissions(['orbital.read', 'orbital.manage'])
    api.fetchOrbitalEvents
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce([{ id: 'evt-1', event_type: 'maneuver', event_time: '2026-08-30T18:00:00Z', source: 'Operator', confidence: 0.9, severity: 'medium' }])
    api.createOrbitalEvent.mockResolvedValue({ id: 'evt-1' })
    renderPanel()

    fireEvent.change(screen.getByLabelText(/^event time$/i), { target: { value: '2026-08-30T18:00' } })
    fireEvent.change(screen.getByLabelText(/^source$/i), { target: { value: ' Operator ' } })
    fireEvent.change(screen.getByLabelText(/^confidence$/i), { target: { value: '0.9' } })
    fireEvent.change(screen.getByLabelText(/^severity$/i), { target: { value: 'medium' } })
    fireEvent.click(screen.getByRole('button', { name: /add orbital event/i }))

    await waitFor(() => expect(api.createOrbitalEvent).toHaveBeenCalledWith('sat-1', expect.objectContaining({
      eventType: 'maneuver',
      source: 'Operator',
      confidence: 0.9,
      severity: 'medium',
      eventTime: expect.stringMatching(/^2026-08-30T/),
    })))
    expect(await screen.findByText('Operator')).toBeInTheDocument()
  })

  it('does not expose orbital operations without orbital capabilities', () => {
    setEffectivePermissions([])
    renderPanel()
    expect(screen.getByText(/do not have permission to view orbital operations/i)).toBeInTheDocument()
    expect(api.fetchOrbitalEvents).not.toHaveBeenCalled()
  })
})
