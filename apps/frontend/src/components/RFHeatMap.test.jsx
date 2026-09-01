import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import RFHeatMap, { buildRFHeatStations } from './RFHeatMap.jsx'

const agents = [
  { agent_id: 'sdr-madrid', station: { latitude_deg: 40.4168, longitude_deg: -3.7038 } },
  { agent_id: 'sdr-paris', station: { latitude_deg: 48.8566, longitude_deg: 2.3522 } },
  { agent_id: 'no-location' },
]

const readings = [
  { agent_id: 'sdr-madrid', frequency_hz: 437_500_000, power_dbm: -70, captured_at: '2026-08-31T20:00:00Z' },
  { agent_id: 'sdr-madrid', frequency_hz: 437_600_000, power_dbm: -50, captured_at: '2026-08-31T20:01:00Z' },
  { agent_id: 'sdr-paris', frequency_hz: 145_800_000, power_dbm: -90, captured_at: '2026-08-31T20:02:00Z' },
]

function renderHeatMap(props = {}) {
  return render(<I18nProvider><RFHeatMap agents={agents} readings={readings} {...props} /></I18nProvider>)
}

describe('RFHeatMap', () => {
  beforeEach(() => localStorage.clear())

  it('aggregates real RF readings by configured station without geographic interpolation', () => {
    const result = buildRFHeatStations({ agents, readings, now: Date.parse('2026-08-31T20:05:00Z') })
    expect(result.activeCount).toBe(2)
    expect(result.sampleCount).toBe(3)
    const madrid = result.stations.find((station) => station.agentId === 'sdr-madrid')
    expect(madrid.stats.averagePowerDbm).toBe(-60)
    expect(madrid.stats.peakPowerDbm).toBe(-50)
  })

  it('renders station intensity summary and accessible station details', () => {
    renderHeatMap()
    expect(screen.getByText('RF signal heat map')).toBeInTheDocument()
    expect(screen.getByText('Stations: 2/2 active · Samples: 3')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sdr-madrid.*average -60.0 dBm.*peak -50.0 dBm.*2 readings/i })).toBeInTheDocument()
  })

  it('filters by agent and frequency from the UI', async () => {
    const user = userEvent.setup()
    renderHeatMap()

    await user.selectOptions(screen.getByLabelText('Agent'), 'sdr-madrid')
    expect(screen.getByText('Stations: 1/1 active · Samples: 2')).toBeInTheDocument()

    await user.clear(screen.getByLabelText('Minimum frequency (MHz)'))
    await user.type(screen.getByLabelText('Minimum frequency (MHz)'), '437.55')
    expect(screen.getByText('Stations: 1/1 active · Samples: 1')).toBeInTheDocument()
  })

  it('reports an invalid frequency range instead of silently swapping it', async () => {
    const user = userEvent.setup()
    renderHeatMap()

    await user.type(screen.getByLabelText('Minimum frequency (MHz)'), '500')
    await user.type(screen.getByLabelText('Maximum frequency (MHz)'), '100')
    expect(screen.getByRole('alert')).toHaveTextContent('Minimum frequency cannot be greater than maximum frequency.')
  })

  it('renders a clear empty state when no stations have coordinates', () => {
    render(<I18nProvider><RFHeatMap agents={[{ agent_id: 'sdr-1' }]} readings={[]} /></I18nProvider>)
    expect(screen.getByText('No SDR station coordinates configured.')).toBeInTheDocument()
  })
})
