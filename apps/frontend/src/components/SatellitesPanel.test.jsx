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

const satelliteB = {
  id: 'sat-2',
  norad_id: 43700,
  name: 'AQUA',
  tle_line1: '1 43700U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995',
  tle_line2: '2 43700  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754',
  tle_status: 'aging',
  orbital_confidence: 'medium',
  updated_at: '2026-08-30T18:00:00Z',
  position: { latitude_deg: 12.4, longitude_deg: 78.9, altitude_km: 705 },
}

const satelliteNoPosition = {
  id: 'sat-3',
  norad_id: 10000,
  name: 'DERELICT-1',
  tle_line1: '',
  tle_line2: '',
  tle_status: 'unusable',
  orbital_confidence: null,
  updated_at: null,
  position: null,
}

function renderPanel(props = {}) {
  return render(<I18nProvider><SatellitesPanel satellites={[satellite]} {...props} /></I18nProvider>)
}

function namesInOrder() {
  return screen.getAllByRole('row').slice(1).map((row) => row.querySelector('td span')?.textContent)
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

  it('opens the satellite security profile without requiring manage permission', () => {
    const onOpenProfile = vi.fn()
    renderPanel({ canManage: false, onOpenProfile })
    fireEvent.click(screen.getByRole('button', { name: /open security profile for ISS/i }))
    expect(onOpenProfile).toHaveBeenCalledWith(satellite)
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

describe('SatellitesPanel sorting', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('sorts by satellite name ascending then descending on repeated clicks', () => {
    render(<I18nProvider><SatellitesPanel satellites={[satellite, satelliteB]} /></I18nProvider>)
    expect(namesInOrder()).toEqual(['ISS', 'AQUA'])

    fireEvent.click(screen.getByRole('button', { name: /sort by satellite/i }))
    expect(namesInOrder()).toEqual(['AQUA', 'ISS'])
    expect(screen.getByRole('columnheader', { name: /satellite/i })).toHaveAttribute('aria-sort', 'ascending')

    fireEvent.click(screen.getByRole('button', { name: /sort by satellite/i }))
    expect(namesInOrder()).toEqual(['ISS', 'AQUA'])
    expect(screen.getByRole('columnheader', { name: /satellite/i })).toHaveAttribute('aria-sort', 'descending')
  })

  it('sorts by altitude numerically, not lexicographically', () => {
    render(<I18nProvider><SatellitesPanel satellites={[satellite, satelliteB]} /></I18nProvider>)
    fireEvent.click(screen.getByRole('button', { name: /sort by altitude/i }))
    expect(namesInOrder()).toEqual(['ISS', 'AQUA'])
  })

  it('keeps satellites without a position at the end regardless of sort direction', () => {
    render(<I18nProvider><SatellitesPanel satellites={[satellite, satelliteNoPosition, satelliteB]} /></I18nProvider>)
    fireEvent.click(screen.getByRole('button', { name: /sort by altitude/i }))
    expect(namesInOrder()).toEqual(['ISS', 'AQUA', 'DERELICT-1'])

    fireEvent.click(screen.getByRole('button', { name: /sort by altitude/i }))
    expect(namesInOrder()).toEqual(['AQUA', 'ISS', 'DERELICT-1'])
  })

  it('switching sort column resets to ascending order', () => {
    render(<I18nProvider><SatellitesPanel satellites={[satellite, satelliteB]} /></I18nProvider>)
    fireEvent.click(screen.getByRole('button', { name: /sort by satellite/i }))
    fireEvent.click(screen.getByRole('button', { name: /sort by satellite/i }))
    fireEvent.click(screen.getByRole('button', { name: /sort by norad id/i }))
    expect(namesInOrder()).toEqual(['ISS', 'AQUA'])
    expect(screen.getByRole('columnheader', { name: /norad id/i })).toHaveAttribute('aria-sort', 'ascending')
  })
})
