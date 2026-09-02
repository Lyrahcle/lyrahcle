import { describe, expect, it } from 'vitest'
import { buildCorrelationGraph, type CorrelationGraph } from './correlationGraph'
import {
  DEFAULT_RING_RADII,
  buildOrbitalLayout,
  edgeKey,
  linkControlPoint,
  pathEdgeKeys,
  quadraticPoints,
} from './correlationLayout3d'

const now = Date.parse('2026-09-02T00:00:00Z')

function connectedGraph(): CorrelationGraph {
  return buildCorrelationGraph({
    satellites: [{ id: 'sat-1', name: 'LYRA-1' }, { id: 'sat-2', name: 'LYRA-2' }],
    agents: [{ agent_id: 'sdr-madrid', station: { name: 'Madrid GS', latitude_deg: 40.4, longitude_deg: -3.7 } }],
    readings: [{ id: 'read-1', agent_id: 'sdr-madrid', satellite_id: 'sat-1', captured_at: '2026-09-01T23:00:00Z' }],
    anomalies: [{ id: 'rf-1', agent_id: 'sdr-madrid', satellite_id: 'sat-1', z_score: 6.2, detected_at: '2026-09-01T23:10:00Z' }],
    incidents: { items: [{ id: 'inc-1', satellite_id: 'sat-1', title: 'Anomaly review', severity: 'high', created_at: '2026-09-01T23:20:00Z' }] },
    now,
    windowMs: null,
  })
}

function isolatedGraph(count: number): CorrelationGraph {
  return buildCorrelationGraph({
    satellites: Array.from({ length: count }, (_, index) => ({ id: `sat-${index}`, name: `SAT-${index}` })),
    now,
    windowMs: null,
  })
}

function distance(a: { x: number; y: number; z: number }, b: { x: number; y: number; z: number }): number {
  return Math.sqrt((a.x - b.x) ** 2 + (a.y - b.y) ** 2 + (a.z - b.z) ** 2)
}

function radius(point: { x: number; y: number; z: number }): number {
  return Math.sqrt(point.x * point.x + point.y * point.y + point.z * point.z)
}

describe('buildOrbitalLayout', () => {
  it('pins the focus node at the scene origin', () => {
    const graph = connectedGraph()
    const visible = new Set(graph.nodes.map((node) => node.id))
    const layout = buildOrbitalLayout(graph, 'satellite:sat-1', visible)
    const focus = layout.byId.get('satellite:sat-1')
    expect(focus).toBeDefined()
    expect(focus?.ring).toBe(0)
    expect(radius(focus!.position)).toBeCloseTo(0, 5)
  })

  it('spreads unlinked nodes across the shells instead of collapsing them', () => {
    const graph = isolatedGraph(18)
    const visible = new Set(graph.nodes.map((node) => node.id))
    const layout = buildOrbitalLayout(graph, 'satellite:sat-0', visible)
    expect(layout.linkCount).toBe(0)
    expect(layout.placements).toHaveLength(18)
    const orbiting = layout.placements.filter((placement) => placement.ring > 0)
    expect(orbiting.length).toBe(17)
    for (const placement of orbiting) expect(radius(placement.position)).toBeGreaterThan(1)
    const unique = new Set(orbiting.map((placement) => `${placement.position.x.toFixed(3)}:${placement.position.z.toFixed(3)}`))
    expect(unique.size).toBe(orbiting.length)
  })

  it('places closer evidence on inner shells', () => {
    const graph = connectedGraph()
    const visible = new Set(graph.nodes.map((node) => node.id))
    const layout = buildOrbitalLayout(graph, 'satellite:sat-1', visible)
    const station = layout.byId.get('station:sdr-madrid')
    const disconnected = layout.byId.get('satellite:sat-2')
    expect(station?.ring).toBe(1)
    expect(disconnected?.ring).toBe(DEFAULT_RING_RADII.length - 1)
    expect(radius(station!.position)).toBeLessThan(radius(disconnected!.position))
  })

  it('is deterministic for identical inputs', () => {
    const graph = connectedGraph()
    const visible = new Set(graph.nodes.map((node) => node.id))
    const first = buildOrbitalLayout(graph, 'satellite:sat-1', visible)
    const second = buildOrbitalLayout(graph, 'satellite:sat-1', visible)
    for (const placement of first.placements) {
      const other = second.byId.get(placement.node.id)
      expect(other?.position.x).toBeCloseTo(placement.position.x, 10)
      expect(other?.position.y).toBeCloseTo(placement.position.y, 10)
      expect(other?.position.z).toBeCloseTo(placement.position.z, 10)
    }
  })

  it('keeps a minimum separation between orbiting nodes', () => {
    const graph = connectedGraph()
    const visible = new Set(graph.nodes.map((node) => node.id))
    const layout = buildOrbitalLayout(graph, 'satellite:sat-1', visible)
    for (let a = 0; a < layout.placements.length; a += 1) {
      for (let b = a + 1; b < layout.placements.length; b += 1) {
        expect(distance(layout.placements[a].position, layout.placements[b].position)).toBeGreaterThan(1.5)
      }
    }
  })

  it('reports one cluster centroid per visible node kind', () => {
    const graph = connectedGraph()
    const visible = new Set(graph.nodes.map((node) => node.id))
    const layout = buildOrbitalLayout(graph, 'satellite:sat-1', visible)
    const kinds = new Set(layout.placements.map((placement) => placement.node.kind))
    expect(layout.clusters).toHaveLength(kinds.size)
    expect(layout.clusters.reduce((total, cluster) => total + cluster.count, 0)).toBe(layout.placements.length)
  })

  it('honours the visible-node filter', () => {
    const graph = connectedGraph()
    const layout = buildOrbitalLayout(graph, 'satellite:sat-1', new Set(['satellite:sat-1', 'station:sdr-madrid']))
    expect(layout.placements).toHaveLength(2)
  })
})

describe('link geometry helpers', () => {
  it('lifts the control point away from the scene centre', () => {
    const source = { x: 10, y: 0, z: 0 }
    const target = { x: 0, y: 0, z: 10 }
    const control = linkControlPoint(source, target)
    const midRadius = radius({ x: 5, y: 0, z: 5 })
    expect(radius(control)).toBeGreaterThan(midRadius)
  })

  it('produces a curve that starts and ends on its endpoints', () => {
    const source = { x: -6, y: 1, z: 2 }
    const target = { x: 7, y: -2, z: 4 }
    const points = quadraticPoints(source, linkControlPoint(source, target), target, 12)
    expect(points).toHaveLength(13)
    expect(points[0]).toEqual(source)
    expect(points[points.length - 1]).toEqual(target)
  })

  it('derives order-independent edge keys for route highlighting', () => {
    const keys = pathEdgeKeys(['a', 'b', 'c'])
    expect(keys.size).toBe(2)
    expect(keys.has(edgeKey({ id: 'x', source: 'b', target: 'a', relation: 'observed', count: 1, strength: 1 }))).toBe(true)
    expect(keys.has(edgeKey({ id: 'y', source: 'c', target: 'b', relation: 'observed', count: 1, strength: 1 }))).toBe(true)
  })

  it('returns no edge keys for a route shorter than two nodes', () => {
    expect(pathEdgeKeys([]).size).toBe(0)
    expect(pathEdgeKeys(['only']).size).toBe(0)
  })
})
