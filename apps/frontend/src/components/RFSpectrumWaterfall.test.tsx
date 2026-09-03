import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import RFSpectrumWaterfall, { buildWaterfallGrid } from './RFSpectrumWaterfall'

const readings = [
  { agent_id: 'gs-madrid', frequency_hz: 437_500_000, power_dbm: -82, captured_at: '2026-09-02T04:00:10Z' },
  { agent_id: 'gs-madrid', frequency_hz: 437_500_000, power_dbm: -70, captured_at: '2026-09-02T04:00:40Z' },
  { agent_id: 'gs-madrid', frequency_hz: 437_525_000, power_dbm: -64, captured_at: '2026-09-02T04:01:10Z' },
  { agent_id: 'gs-paris', frequency_hz: 145_800_000, power_dbm: -91, captured_at: '2026-09-02T04:01:20Z' },
]

function renderWaterfall() {
  return render(<I18nProvider><RFSpectrumWaterfall readings={readings} /></I18nProvider>)
}

describe('RFSpectrumWaterfall', () => {
  beforeEach(() => localStorage.clear())

  it('builds a bounded waterfall from real samples and keeps the strongest sample in each minute/frequency cell', () => {
    const grid = buildWaterfallGrid(readings)
    expect(grid.rows).toHaveLength(2)
    expect(grid.frequencies).toEqual([145_800_000, 437_500_000, 437_525_000])
    expect(grid.sampleCount).toBe(4)
    expect(grid.maxPowerDbm).toBe(-64)
    const firstRow = grid.rows[0]
    expect(firstRow.cells.find((cell) => cell.frequencyHz === 437_500_000)?.powerDbm).toBe(-70)
  })

  it('renders the RF waterfall with real frequency and power evidence', () => {
    renderWaterfall()
    expect(screen.getByText('RF Spectrum Waterfall')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'RF power over frequency and time' })).toBeInTheDocument()
    expect(screen.getByTitle('437.525000 MHz · -64.0 dBm')).toBeInTheDocument()
    expect(screen.getByText('2 time slices · 3 frequencies · 4 samples')).toBeInTheDocument()
  })

  it('filters the waterfall by SDR agent without fabricating missing samples', async () => {
    const user = userEvent.setup()
    renderWaterfall()
    await user.selectOptions(screen.getByLabelText('Agent'), 'gs-paris')
    expect(screen.getByText('1 time slices · 1 frequencies · 1 samples')).toBeInTheDocument()
    expect(screen.getByTitle('145.800000 MHz · -91.0 dBm')).toBeInTheDocument()
  })
})
