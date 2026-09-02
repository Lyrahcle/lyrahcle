import { fireEvent, render, screen, within } from '@testing-library/react'
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

describe('CorrelationGraph orbital focus', () => {
  it('renders the flagship workspace with a keyboard-reachable node index', () => {
    const { container } = renderGraph()
    expect(screen.getByRole('heading', { name: 'Correlation constellation' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Interactive Lyrahcle correlation constellation' })).toBeInTheDocument()
    expect(screen.getByText('Mission evidence fabric')).toBeInTheDocument()
    const satellite = container.querySelector('[aria-label="Satellite: LYRA-1"]')
    expect(satellite).toHaveAttribute('tabindex', '0')
    expect(satellite).toHaveAttribute('data-correlation-node', 'satellite:sat-1')
  })

  it('degrades to an explainable non-WebGL surface without losing interaction', () => {
    renderGraph()
    expect(screen.getByText(/no WebGL context/i)).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Find a node…')).toBeInTheDocument()
  })

  it('re-orbits on selection and traces an explainable route', () => {
    const { container } = renderGraph()
    fireEvent.click(screen.getByRole('button', { name: 'Trace from here' }))
    const station = container.querySelector('[aria-label="Ground station: Madrid GS"]')
    expect(station).not.toBeNull()
    fireEvent.click(station as Element)
    expect(screen.getByText('Explainable route')).toBeInTheDocument()
    expect(screen.getAllByText(/LYRA-1/).length).toBeGreaterThan(0)
    expect(screen.getAllByText(/Madrid GS/).length).toBeGreaterThan(0)
  })

  it('opens a satellite profile directly from the context panel', () => {
    const { onOpenSatellite } = renderGraph()
    fireEvent.click(screen.getByRole('button', { name: 'Open in workspace' }))
    expect(onOpenSatellite).toHaveBeenCalledWith('sat-1')
  })

  it('offers autocomplete search that recenters the constellation', () => {
    renderGraph()
    const input = screen.getByPlaceholderText('Find a node…')
    fireEvent.change(input, { target: { value: 'CVE-2026' } })
    const listbox = screen.getByRole('listbox', { name: 'Matching nodes' })
    const options = within(listbox).getAllByRole('option')
    expect(options.length).toBeGreaterThan(0)
    expect(options[0]).toHaveAttribute('aria-selected', 'true')
    fireEvent.keyDown(input, { key: 'Enter' })
    expect((input as HTMLInputElement).value).toBe('')
    expect(screen.getAllByText(/CVE-2026-1000/).length).toBeGreaterThan(0)
  })

  it('toggles focus mode and the filter popover from the HUD', () => {
    renderGraph()
    const focusMode = screen.getByRole('button', { name: 'Focus mode' })
    expect(focusMode).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(focusMode)
    expect(focusMode).toHaveAttribute('aria-pressed', 'true')

    fireEvent.click(screen.getByRole('button', { name: 'Filters' }))
    const filters = screen.getByRole('group', { name: 'Filters' })
    expect(within(filters).getByRole('button', { name: 'Critical' })).toBeInTheDocument()
    expect(within(filters).getByRole('button', { name: 'CVE' })).toBeInTheDocument()
  })

  it('toggles cluster markers from the HUD', () => {
    renderGraph()
    const clusters = screen.getByRole('button', { name: 'Clusters' })
    expect(clusters).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(clusters)
    expect(clusters).toHaveAttribute('aria-pressed', 'false')
  })

  it('announces the current orbital focus to assistive technology', () => {
    const { container } = renderGraph()
    const status = container.querySelector('[role="status"][aria-live="polite"]')
    expect(status).toHaveTextContent('Orbital focus: Satellite LYRA-1')

    fireEvent.click(container.querySelector('[aria-label="Ground station: Madrid GS"]') as Element)
    expect(status).toHaveTextContent('Orbital focus: Ground station Madrid GS')
  })

  it('restores every HUD toggle when the view is reset', () => {
    renderGraph()
    fireEvent.click(screen.getByRole('button', { name: 'Focus mode' }))
    fireEvent.click(screen.getByRole('button', { name: 'Clusters' }))
    expect(screen.getByRole('button', { name: 'Focus mode' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Clusters' })).toHaveAttribute('aria-pressed', 'false')

    fireEvent.click(screen.getByRole('button', { name: 'Reset view' }))
    expect(screen.getByRole('button', { name: 'Focus mode' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Clusters' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('keeps a populated constellation when nodes have no links yet', () => {
    const { container } = renderGraph({
      satellites: Array.from({ length: 21 }, (_, index) => ({ id: `sat-${index + 1}`, name: `SAT-${String(index + 1).padStart(2, '0')}` })),
      agents: [], readings: [], anomalies: [], incidents: { items: [] }, assets: { items: [] }, programs: [], alerts: { items: [] }, cves: { items: [] },
    })
    expect(container.querySelectorAll('[data-correlation-node]')).toHaveLength(21)
    expect(screen.getByText(/21 nodes · 0 links/)).toBeInTheDocument()
  })
})
