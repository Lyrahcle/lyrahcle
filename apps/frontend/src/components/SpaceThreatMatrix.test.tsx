import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import SpaceThreatMatrix, { buildSpaceThreatMatrix } from './SpaceThreatMatrix'

describe('SpaceThreatMatrix', () => {
  it('maps only evidence-backed RF candidates into the matrix', () => {
    const cells = buildSpaceThreatMatrix([
      { id: 'a1', agent_id: 'gs-madrid', metric: 'power_dbm', z_score: 6.2 },
      { id: 'a2', agent_id: 'gs-paris', metric: 'frequency_hz', z_score: -5.1 },
      { id: 'a3', agent_id: 'gs-rome', metric: 'power_dbm', z_score: 2.0 },
    ], [])

    expect(cells.find((cell) => cell.id === 'rf-interference')?.evidenceCount).toBe(1)
    expect(cells.find((cell) => cell.id === 'signal-spoofing')?.evidenceCount).toBe(1)
    expect(cells.find((cell) => cell.id === 'replay')?.evidenceCount).toBe(0)
  })

  it('uses explicit incident wording for ground access and exploitation evidence', () => {
    const cells = buildSpaceThreatMatrix([], [
      { id: 'i1', title: 'Unauthorized ground station access attempt' },
      { id: 'i2', title: 'CVE-2026-1234 exploitation detected' },
      { id: 'i3', title: 'High temperature warning' },
    ])

    expect(cells.find((cell) => cell.id === 'ground-access')?.evidenceCount).toBe(1)
    expect(cells.find((cell) => cell.id === 'software-exploitation')?.evidenceCount).toBe(1)
  })

  it('renders unavailable techniques without claiming attacks that lack evidence', () => {
    render(
      <I18nProvider>
        <SpaceThreatMatrix anomalies={[]} incidents={[]} />
      </I18nProvider>,
    )

    expect(screen.getByText('Space threat matrix')).toBeInTheDocument()
    expect(screen.getByText('Replay / repeated command')).toBeInTheDocument()
    expect(screen.getAllByText('Evidence unavailable').length).toBeGreaterThan(0)
  })
})
