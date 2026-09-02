import { describe, expect, it } from 'vitest'
import { buildCorrelationGraph, chooseCorrelationAnchor, correlationNeighborhood, orbitLayout3D, shortestCorrelationPath } from './correlationGraph'

const now = Date.parse('2026-09-02T00:00:00Z')

function sampleGraph() {
  return buildCorrelationGraph({
    now,
    windowMs: 7 * 24 * 60 * 60 * 1000,
    satellites: [{ id: 'sat-1', name: 'LYRA-1', norad_id: 12345 }],
    agents: [{ agent_id: 'sdr-madrid', station: { name: 'Madrid GS', latitude_deg: 40.4, longitude_deg: -3.7 } }],
    readings: [{ id: 'read-1', agent_id: 'sdr-madrid', satellite_id: 'sat-1', captured_at: '2026-09-01T23:00:00Z' }],
    anomalies: [{ id: 'rf-1', agent_id: 'sdr-madrid', satellite_id: 'sat-1', metric: 'power_dbm', z_score: 6.2, detected_at: '2026-09-01T23:10:00Z' }],
    incidents: { items: [{ id: 'inc-1', satellite_id: 'sat-1', agent_id: 'sdr-madrid', title: 'CVE-2026-1000 uplink investigation', severity: 'critical', created_at: '2026-09-01T23:20:00Z' }] },
    programs: [{ id: 'program-1', name: 'Flight segment' }],
    assets: { items: [{ id: 'asset-1', program_id: 'program-1', kind: 'satellite', value: 'LYRA-1' }] },
    alerts: { items: [{ id: 'alert-1', asset_value: 'LYRA-1', cve_id: 'CVE-2026-1000', severity: 'high', created_at: '2026-09-01T23:30:00Z' }] },
    cves: { items: [{ cve_id: 'CVE-2026-1000', severity: 'high', summary: 'Flight software exposure' }] },
  })
}

describe('correlation graph model', () => {
  it('builds cross-domain links from existing Lyrahcle identifiers', () => {
    const graph = sampleGraph()
    expect(graph.nodes.some((node) => node.id === 'satellite:sat-1')).toBe(true)
    expect(graph.nodes.some((node) => node.id === 'station:sdr-madrid')).toBe(true)
    expect(graph.nodes.some((node) => node.id === 'cve:CVE-2026-1000')).toBe(true)
    expect(graph.edges.some((edge) => edge.relation === 'observed' && edge.count === 1)).toBe(true)
    expect(graph.edges.some((edge) => edge.relation === 'exposed')).toBe(true)
    expect(graph.edges.some((edge) => edge.relation === 'mentioned')).toBe(true)
  })

  it('traces the shortest explainable path through the evidence fabric', () => {
    const graph = sampleGraph()
    const path = shortestCorrelationPath(graph, 'station:sdr-madrid', 'cve:CVE-2026-1000')
    expect(path[0]).toBe('station:sdr-madrid')
    expect(path.at(-1)).toBe('cve:CVE-2026-1000')
    expect(path.length).toBeGreaterThanOrEqual(3)
  })

  it('uses a satellite as the preferred default orbital focus', () => {
    expect(chooseCorrelationAnchor(sampleGraph())).toBe('satellite:sat-1')
  })

  it('projects the same evidence fabric differently as the camera orbits in 3D', () => {
    const graph = sampleGraph()
    const visible = new Set(graph.nodes.map((node) => node.id))
    const first = orbitLayout3D(graph, 'satellite:sat-1', visible, { yaw: -0.4, pitch: -0.15 })
    const second = orbitLayout3D(graph, 'satellite:sat-1', visible, { yaw: 0.5, pitch: 0.12 })
    const firstStation = first.find((point) => point.node.id === 'station:sdr-madrid')
    const secondStation = second.find((point) => point.node.id === 'station:sdr-madrid')
    expect(firstStation).toBeDefined()
    expect(secondStation).toBeDefined()
    expect(firstStation?.x).not.toBe(secondStation?.x)
    expect(firstStation?.z).not.toBe(secondStation?.z)
  })

  it('limits orbital focus to the selected graph distance', () => {
    const graph = sampleGraph()
    const oneHop = correlationNeighborhood(graph, 'station:sdr-madrid', 1)
    const twoHop = correlationNeighborhood(graph, 'station:sdr-madrid', 2)
    expect(oneHop.has('station:sdr-madrid')).toBe(true)
    expect(twoHop.size).toBeGreaterThanOrEqual(oneHop.size)
  })

  it('excludes stale evidence while retaining mission topology', () => {
    const graph = buildCorrelationGraph({
      now,
      windowMs: DAY,
      satellites: [{ id: 'sat-1', name: 'LYRA-1' }],
      anomalies: [{ id: 'old', satellite_id: 'sat-1', metric: 'power_dbm', z_score: 8, detected_at: '2026-08-20T00:00:00Z' }],
    })
    expect(graph.nodes.some((node) => node.kind === 'satellite')).toBe(true)
    expect(graph.nodes.some((node) => node.kind === 'rf')).toBe(false)
  })
})

const DAY = 24 * 60 * 60 * 1000
