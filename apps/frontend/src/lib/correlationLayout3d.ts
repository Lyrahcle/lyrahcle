import {
  correlationDistances,
  graphDegrees,
  type CorrelationEdge,
  type CorrelationGraph,
  type CorrelationNode,
  type CorrelationNodeKind,
} from './correlationGraph'

export interface Vec3 {
  x: number
  y: number
  z: number
}

export interface LayoutPlacement {
  node: CorrelationNode
  ring: number
  cluster: CorrelationNodeKind
  position: Vec3
  degree: number
}

export interface ClusterCentroid {
  kind: CorrelationNodeKind
  position: Vec3
  count: number
}

export interface OrbitalLayout {
  placements: LayoutPlacement[]
  byId: Map<string, LayoutPlacement>
  clusters: ClusterCentroid[]
  ringRadii: number[]
  extent: number
  linkCount: number
}

export interface OrbitalLayoutOptions {
  ringRadii?: number[]
  relaxIterations?: number
  clusterStrength?: number
  linkRest?: number
  separation?: number
}

export const LAYOUT_KIND_ORDER: CorrelationNodeKind[] = [
  'satellite',
  'station',
  'agent',
  'incident',
  'rf',
  'asset',
  'cve',
  'program',
]

export const DEFAULT_RING_RADII = [0, 15, 27, 39]

const TAU = Math.PI * 2
const GOLDEN_ANGLE = Math.PI * (3 - Math.sqrt(5))

function hashUnit(value: string, salt: number): number {
  let hash = 2166136261 ^ salt
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return ((hash >>> 0) % 100000) / 100000
}

function clamp(value: number, min: number, max: number): number {
  return value < min ? min : value > max ? max : value
}

function length(vector: Vec3): number {
  return Math.sqrt(vector.x * vector.x + vector.y * vector.y + vector.z * vector.z)
}

function sectorCenter(kind: CorrelationNodeKind): number {
  const index = Math.max(0, LAYOUT_KIND_ORDER.indexOf(kind))
  return (TAU * index) / LAYOUT_KIND_ORDER.length
}

function ringElevation(ring: number): number {
  if (ring <= 1) return 0.34
  if (ring === 2) return 0.46
  return 0.56
}

export function shellPosition(
  node: CorrelationNode,
  ring: number,
  radius: number,
  indexInCluster: number,
  clusterSize: number,
  clusterStrength: number,
): Vec3 {
  const spread = (TAU / LAYOUT_KIND_ORDER.length) * (1 + (1 - clamp(clusterStrength, 0, 1)) * 4.2)
  const slot = clusterSize > 1 ? (indexInCluster + 0.5) / clusterSize - 0.5 : 0
  const jitter = (hashUnit(node.id, ring * 31 + 7) - 0.5) * 0.22
  const azimuth = sectorCenter(node.kind) + slot * spread + jitter + indexInCluster * GOLDEN_ANGLE * 0.08
  const elevationBand = ringElevation(ring)
  const elevationSlot = clusterSize > 1 ? (indexInCluster % 5) / 4 - 0.5 : 0
  const elevation =
    elevationSlot * elevationBand + (hashUnit(node.id, ring * 17 + 3) - 0.5) * elevationBand * 0.5
  const horizontal = Math.cos(elevation) * radius
  return {
    x: Math.cos(azimuth) * horizontal,
    y: Math.sin(elevation) * radius * 0.62,
    z: Math.sin(azimuth) * horizontal,
  }
}

function relax(
  placements: LayoutPlacement[],
  byId: Map<string, LayoutPlacement>,
  edges: CorrelationEdge[],
  ringRadii: number[],
  options: Required<Pick<OrbitalLayoutOptions, 'relaxIterations' | 'linkRest' | 'separation'>>,
): void {
  if (!edges.length || placements.length < 3) return
  const movable = placements.filter((placement) => placement.ring > 0)
  if (!movable.length) return

  for (let step = 0; step < options.relaxIterations; step += 1) {
    const cooling = 1 - step / options.relaxIterations
    const forces = new Map<string, Vec3>()
    for (const placement of movable) forces.set(placement.node.id, { x: 0, y: 0, z: 0 })

    for (const edge of edges) {
      const source = byId.get(edge.source)
      const target = byId.get(edge.target)
      if (!source || !target) continue
      const dx = target.position.x - source.position.x
      const dy = target.position.y - source.position.y
      const dz = target.position.z - source.position.z
      const distance = Math.max(0.001, Math.sqrt(dx * dx + dy * dy + dz * dz))
      const pull = ((distance - options.linkRest) / distance) * 0.09 * Math.min(2, edge.strength + 0.4)
      const sourceForce = forces.get(source.node.id)
      const targetForce = forces.get(target.node.id)
      if (sourceForce) {
        sourceForce.x += dx * pull
        sourceForce.y += dy * pull
        sourceForce.z += dz * pull
      }
      if (targetForce) {
        targetForce.x -= dx * pull
        targetForce.y -= dy * pull
        targetForce.z -= dz * pull
      }
    }

    for (let a = 0; a < placements.length; a += 1) {
      for (let b = a + 1; b < placements.length; b += 1) {
        const first = placements[a]
        const second = placements[b]
        const dx = second.position.x - first.position.x
        const dy = second.position.y - first.position.y
        const dz = second.position.z - first.position.z
        const distance = Math.sqrt(dx * dx + dy * dy + dz * dz)
        if (distance >= options.separation || distance < 0.0001) continue
        const push = ((options.separation - distance) / distance) * 0.28
        const firstForce = forces.get(first.node.id)
        const secondForce = forces.get(second.node.id)
        if (firstForce) {
          firstForce.x -= dx * push
          firstForce.y -= dy * push
          firstForce.z -= dz * push
        }
        if (secondForce) {
          secondForce.x += dx * push
          secondForce.y += dy * push
          secondForce.z += dz * push
        }
      }
    }

    for (const placement of movable) {
      const force = forces.get(placement.node.id)
      if (!force) continue
      placement.position.x += force.x * cooling
      placement.position.y += force.y * cooling
      placement.position.z += force.z * cooling
      const target = ringRadii[Math.min(placement.ring, ringRadii.length - 1)]
      const current = length(placement.position)
      if (current < 0.0001) {
        placement.position.x = target
        continue
      }
      const corrected = current + (target - current) * 0.55
      const factor = corrected / current
      placement.position.x *= factor
      placement.position.y *= factor
      placement.position.z *= factor
    }
  }
}

export function buildOrbitalLayout(
  graph: CorrelationGraph,
  focusId: string | null,
  visibleIds: Set<string>,
  options: OrbitalLayoutOptions = {},
): OrbitalLayout {
  const ringRadii = options.ringRadii ?? DEFAULT_RING_RADII
  const clusterStrength = options.clusterStrength ?? 0.62
  const distances = focusId ? correlationDistances(graph, focusId) : new Map<string, number>()
  const degrees = graphDegrees(graph)
  const visible = graph.nodes.filter((node) => visibleIds.has(node.id))
  const grouped = new Map<string, CorrelationNode[]>()

  for (const node of visible) {
    if (focusId && node.id === focusId) continue
    const reach = distances.get(node.id)
    const ring = clamp(Number.isFinite(reach) ? (reach as number) : ringRadii.length - 1, 1, ringRadii.length - 1)
    const key = `${ring}:${node.kind}`
    const bucket = grouped.get(key) ?? []
    bucket.push(node)
    grouped.set(key, bucket)
  }

  const placements: LayoutPlacement[] = []
  const focusNode = focusId ? visible.find((node) => node.id === focusId) : undefined
  if (focusNode) {
    placements.push({
      node: focusNode,
      ring: 0,
      cluster: focusNode.kind,
      position: { x: 0, y: 0, z: 0 },
      degree: degrees.get(focusNode.id) ?? 0,
    })
  }

  for (const [key, bucket] of [...grouped.entries()].sort((a, b) => a[0].localeCompare(b[0]))) {
    const ring = Number(key.split(':')[0])
    const radius = ringRadii[Math.min(ring, ringRadii.length - 1)]
    const sorted = [...bucket].sort(
      (a, b) => (degrees.get(b.id) ?? 0) - (degrees.get(a.id) ?? 0) || a.label.localeCompare(b.label),
    )
    sorted.forEach((node, index) => {
      placements.push({
        node,
        ring,
        cluster: node.kind,
        position: shellPosition(node, ring, radius, index, sorted.length, clusterStrength),
        degree: degrees.get(node.id) ?? 0,
      })
    })
  }

  const byId = new Map(placements.map((placement) => [placement.node.id, placement]))
  const visibleEdges = graph.edges.filter((edge) => byId.has(edge.source) && byId.has(edge.target))

  relax(placements, byId, visibleEdges, ringRadii, {
    relaxIterations: options.relaxIterations ?? 110,
    linkRest: options.linkRest ?? 11,
    separation: options.separation ?? 5.4,
  })

  const clusterTotals = new Map<CorrelationNodeKind, { x: number; y: number; z: number; count: number }>()
  for (const placement of placements) {
    const current = clusterTotals.get(placement.cluster) ?? { x: 0, y: 0, z: 0, count: 0 }
    current.x += placement.position.x
    current.y += placement.position.y
    current.z += placement.position.z
    current.count += 1
    clusterTotals.set(placement.cluster, current)
  }

  const clusters: ClusterCentroid[] = [...clusterTotals.entries()].map(([kind, total]) => ({
    kind,
    count: total.count,
    position: { x: total.x / total.count, y: total.y / total.count, z: total.z / total.count },
  }))

  const extent = placements.reduce((max, placement) => Math.max(max, length(placement.position)), 1)

  return { placements, byId, clusters, ringRadii, extent, linkCount: visibleEdges.length }
}

export function linkControlPoint(source: Vec3, target: Vec3, lift = 0.16): Vec3 {
  const midX = (source.x + target.x) / 2
  const midY = (source.y + target.y) / 2
  const midZ = (source.z + target.z) / 2
  const span = Math.sqrt(
    (target.x - source.x) ** 2 + (target.y - source.y) ** 2 + (target.z - source.z) ** 2,
  )
  const radial = Math.max(0.001, Math.sqrt(midX * midX + midY * midY + midZ * midZ))
  const push = 1 + (span * lift) / radial
  return { x: midX * push, y: midY * push + span * lift * 0.32, z: midZ * push }
}

export function quadraticPoints(source: Vec3, control: Vec3, target: Vec3, segments = 18): Vec3[] {
  const points: Vec3[] = []
  for (let step = 0; step <= segments; step += 1) {
    const t = step / segments
    const inverse = 1 - t
    points.push({
      x: inverse * inverse * source.x + 2 * inverse * t * control.x + t * t * target.x,
      y: inverse * inverse * source.y + 2 * inverse * t * control.y + t * t * target.y,
      z: inverse * inverse * source.z + 2 * inverse * t * control.z + t * t * target.z,
    })
  }
  return points
}

export function pathEdgeKeys(path: string[]): Set<string> {
  const keys = new Set<string>()
  for (let index = 0; index < path.length - 1; index += 1) {
    const a = path[index]
    const b = path[index + 1]
    keys.add(a < b ? `${a}::${b}` : `${b}::${a}`)
  }
  return keys
}

export function edgeKey(edge: CorrelationEdge): string {
  return edge.source < edge.target ? `${edge.source}::${edge.target}` : `${edge.target}::${edge.source}`
}
