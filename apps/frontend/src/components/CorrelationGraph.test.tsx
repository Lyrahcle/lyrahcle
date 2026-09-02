import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import CorrelationGraph from './CorrelationGraph'

const now = Date.parse('2026-09-02T00:00:00Z')

function renderGraph(overrides = {}) {
  const onNavigate = vi.fn()
  const onOpenSatellite = vi.fn()
  const result = render(
    <I18nProvider>
      <CorrelationGraph
        now={now}
        satellites={[{ id: 'sat-1', name: 'LYRA-1', norad_id: 12345 }]}
        agents={[{ agent_id: 'sdr-madrid', station: { name: 'Madrid GS', latitude_deg: 40.4, longitude_deg: -3.7 } }]}
        readings={[{ id: 'read-1', agent_id: 'sdr-madrid', satellite_id: 'sat-1', captured_at: '2026-09-01T23:00:00Z' }]}
        anomalies={[{ id: 'rf-1', agent_id: 'sdr-madrid', satellite_id: 'sat-1', metric: 'power_dbm', z_score: 6.2, detected_at: '2026-09-01T23:10:00Z' }]}
        incidents={{ items: [{ id: 'inc-1', satellite_id: 'sat-1', agent_id: 'sdr-madrid', title: 'CVE-2026-1000 investigation', severity: 'critical', created_at: '2026-09-01T23:20:00Z' }] }}
        assets={{ items: [{ id: 'asset-1', kind: 'satellite', value: 'LYRA-1', program_id: 'program-1' }] }}
        programs={[{ id: 'program-1', name: 'Flight segment' }]}
        alerts={{ items: [{ id: 'alert-1', asset_value: 'LYRA-1', cve_id: 'CVE-2026-1000', severity: 'high', created_at: '2026-09-01T23:30:00Z' }] }}
        cves={{ items: [{ cve_id: 'CVE-2026-1000', summary: 'Flight software exposure', severity: 'high' }] }}
        access={{ orbital: true, agents: true, rf: true, incidents: true, assets: true, programs: true, alerts: true }}
        onNavigate={onNavigate}
        onOpenSatellite={onOpenSatellite}
        {...overrides}
      />
    </I18nProvider>,
  )
  return { ...result, onNavigate, onOpenSatellite }
}

beforeEach(() => {
  localStorage.setItem('lyra_lang', 'en')
})

describe('CorrelationGraph', () => {
  it('renders the flagship constellation with accessible interactive nodes', () => {
    const { container } = renderGraph()
    expect(screen.getByRole('heading', { name: 'Correlation constellation' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Interactive Lyrahcle correlation constellation' })).toBeInTheDocument()
    expect(container.querySelector('[aria-label="Satellite: LYRA-1"]')).toHaveAttribute('tabindex', '0')
    expect(screen.getByText('Mission evidence fabric')).toBeInTheDocument()
  })

  it('re-orbits on selection and traces an explainable path', () => {
    const { container } = renderGraph()
    fireEvent.click(screen.getByRole('button', { name: 'Trace from here' }))
    const station = container.querySelector('[aria-label="Ground station: Madrid GS"]')
    expect(station).not.toBeNull()
    fireEvent.click(station as Element)
    expect(screen.getByText('Explainable route')).toBeInTheDocument()
    expect(screen.getAllByText(/LYRA-1/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/Madrid GS/).length).toBeGreaterThan(0)
  })

  it('opens a satellite profile directly from graph context', () => {
    const { onOpenSatellite } = renderGraph()
    fireEvent.click(screen.getByRole('button', { name: 'Open in workspace' }))
    expect(onOpenSatellite).toHaveBeenCalledWith('sat-1')
  })

  it('searches nodes without discarding the current evidence fabric', () => {
    const { container } = renderGraph()
    const nodesBeforeSearch = container.querySelectorAll('g[role="button"]').length
    fireEvent.change(screen.getByPlaceholderText('Find a node…'), { target: { value: 'CVE-2026' } })
    expect(screen.getAllByRole('button', { name: /CVE-2026-1000/ }).length).toBeGreaterThan(0)
    expect(container.querySelectorAll('g[role="button"]')).toHaveLength(nodesBeforeSearch)
  })
  it('keeps a visible 3D constellation when nodes have no links yet', () => {
    const { container } = renderGraph({
      satellites: Array.from({ length: 21 }, (_, index) => ({ id: `sat-${index + 1}`, name: `SAT-${String(index + 1).padStart(2, '0')}` })),
      agents: [], readings: [], anomalies: [], incidents: { items: [] }, assets: { items: [] }, programs: [], alerts: { items: [] }, cves: { items: [] },
    })
    expect(container.querySelectorAll('g[role="button"]')).toHaveLength(21)
    expect(screen.getByText(/21 nodes · 0 links/)).toBeInTheDocument()
  })

})
