export type CorrelationNodeKind = 'satellite' | 'station' | 'agent' | 'incident' | 'rf' | 'asset' | 'cve' | 'program'
export type CorrelationSeverity = 'critical' | 'high' | 'medium' | 'low' | 'unknown'

export type CorrelationRecord = Record<string, any>

export interface CorrelationNode {
  id: string
  kind: CorrelationNodeKind
  label: string
  subtitle?: string
  severity: CorrelationSeverity
  timestamp?: string
  entityId?: string
  target: 'orbital' | 'stations' | 'connectivity' | 'incidents' | 'signal' | 'assets' | 'intel'
  metadata: CorrelationRecord
}

export interface CorrelationEdge {
  id: string
  source: string
  target: string
  relation: 'observed' | 'detected' | 'affected' | 'assigned' | 'contains' | 'exposed' | 'matched' | 'mentioned'
  count: number
  strength: number
}

export interface CorrelationGraph {
  nodes: CorrelationNode[]
  edges: CorrelationEdge[]
  evidenceCount: number
  staticCount: number
}

export interface CorrelationGraphInput {
  satellites?: CorrelationRecord[] | null
  agents?: CorrelationRecord[] | null
  readings?: CorrelationRecord[] | null
  anomalies?: CorrelationRecord[] | null
  incidents?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  assets?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  programs?: CorrelationRecord[] | null
  alerts?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  cves?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  now?: number
  windowMs?: number | null
}

export interface CorrelationPoint {
  node: CorrelationNode
  x: number
  y: number
  z: number
  scale: number
  distance: number
}

export interface CorrelationCamera {
  yaw: number
  pitch: number
}

function asItems(value: CorrelationGraphInput['incidents']): CorrelationRecord[] {
  if (Array.isArray(value)) return value
  return Array.isArray(value?.items) ? value.items : []
}

function normalized(value: unknown): string {
  return String(value ?? '').trim().toLocaleLowerCase()
}

function severity(value: unknown): CorrelationSeverity {
  const candidate = normalized(value)
  if (candidate === 'critical' || candidate === 'high' || candidate === 'medium' || candidate === 'low') return candidate
  return 'unknown'
}

function anomalySeverity(item: CorrelationRecord): CorrelationSeverity {
  const z = Math.abs(Number(item?.z_score) || 0)
  if (z >= 7) return 'critical'
  if (z >= 5) return 'high'
  if (z >= 3) return 'medium'
  return 'low'
}

function timestamp(item: CorrelationRecord, fields: string[]): string | undefined {
  for (const field of fields) {
    const value = item?.[field]
    if (value && Number.isFinite(Date.parse(value))) return value
  }
  return undefined
}

function inWindow(item: CorrelationRecord, fields: string[], now: number, windowMs: number | null): boolean {
  if (windowMs === null) return true
  const value = timestamp(item, fields)
  if (!value) return true
  return now - Date.parse(value) <= windowMs
}

function idFor(kind: CorrelationNodeKind, value: unknown): string {
  return `${kind}:${String(value ?? '').trim()}`
}

function displayMetric(value: unknown): string {
  const metric = String(value ?? 'RF').replaceAll('_', ' ')
  return metric.charAt(0).toUpperCase() + metric.slice(1)
}

export function buildCorrelationGraph(input: CorrelationGraphInput): CorrelationGraph {
  const now = input.now ?? Date.now()
  const windowMs = input.windowMs ?? 7 * 24 * 60 * 60 * 1000
  const nodes = new Map<string, CorrelationNode>()
  const edges = new Map<string, CorrelationEdge>()
  const satelliteById = new Map<string, CorrelationNode>()
  const satelliteByValue = new Map<string, CorrelationNode>()
  const agentById = new Map<string, CorrelationNode>()
  const assetByValue = new Map<string, CorrelationNode>()
  const programById = new Map<string, CorrelationNode>()
  const cveByValue = new Map<string, CorrelationRecord>()
  let evidenceCount = 0

  function addNode(node: CorrelationNode): CorrelationNode {
    const current = nodes.get(node.id)
    if (current) return current
    nodes.set(node.id, node)
    return node
  }

  function addEdge(source: string | undefined, target: string | undefined, relation: CorrelationEdge['relation'], strength = 1, count = 1) {
    if (!source || !target || source === target || !nodes.has(source) || !nodes.has(target)) return
    const id = `${source}|${relation}|${target}`
    const current = edges.get(id)
    if (current) {
      current.count += count
      current.strength = Math.max(current.strength, strength)
      return
    }
    edges.set(id, { id, source, target, relation, count, strength })
  }

  for (const cve of asItems(input.cves)) {
    const key = normalized(cve.cve_id || cve.id)
    if (key) cveByValue.set(key, cve)
  }

  for (const satellite of input.satellites ?? []) {
    if (!satellite?.id) continue
    const node = addNode({
      id: idFor('satellite', satellite.id),
      kind: 'satellite',
      label: satellite.name || satellite.norad_id || satellite.id,
      subtitle: satellite.norad_id ? `NORAD ${satellite.norad_id}` : undefined,
      severity: 'unknown',
      timestamp: timestamp(satellite, ['updated_at', 'created_at']),
      entityId: satellite.id,
      target: 'orbital',
      metadata: satellite,
    })
    satelliteById.set(String(satellite.id), node)
    for (const value of [satellite.name, satellite.norad_id, satellite.id]) {
      const key = normalized(value)
      if (key) satelliteByValue.set(key, node)
    }
  }

  for (const agent of input.agents ?? []) {
    if (!agent?.agent_id) continue
    const kind: CorrelationNodeKind = agent.station?.latitude_deg !== undefined && agent.station?.longitude_deg !== undefined ? 'station' : 'agent'
    const node = addNode({
      id: idFor(kind, agent.agent_id),
      kind,
      label: agent.station?.name || agent.agent_id,
      subtitle: agent.station?.name ? agent.agent_id : agent.device?.hardware,
      severity: normalized(agent.status) === 'quarantined' || normalized(agent.status) === 'revoked' ? 'high' : 'unknown',
      timestamp: timestamp(agent, ['last_seen_at', 'updated_at']),
      entityId: agent.agent_id,
      target: kind === 'station' ? 'stations' : 'connectivity',
      metadata: agent,
    })
    agentById.set(String(agent.agent_id), node)
  }

  for (const program of input.programs ?? []) {
    if (!program?.id) continue
    const node = addNode({
      id: idFor('program', program.id),
      kind: 'program',
      label: program.name || program.id,
      subtitle: program.description,
      severity: 'unknown',
      timestamp: timestamp(program, ['updated_at', 'created_at']),
      entityId: program.id,
      target: 'assets',
      metadata: program,
    })
    programById.set(String(program.id), node)
  }

  for (const asset of asItems(input.assets)) {
    if (!asset?.id) continue
    const node = addNode({
      id: idFor('asset', asset.id),
      kind: 'asset',
      label: asset.value || asset.name || asset.id,
      subtitle: asset.kind,
      severity: 'unknown',
      timestamp: timestamp(asset, ['updated_at', 'created_at']),
      entityId: asset.id,
      target: 'assets',
      metadata: asset,
    })
    const valueKey = normalized(asset.value || asset.name)
    if (valueKey) assetByValue.set(valueKey, node)
    addEdge(programById.get(String(asset.program_id))?.id, node.id, 'contains', 0.65)
    const satellite = satelliteByValue.get(valueKey)
    if (satellite) addEdge(satellite.id, node.id, 'matched', 0.8)
  }

  const observationPairs = new Map<string, { station: string; satellite: string; count: number }>()
  for (const reading of input.readings ?? []) {
    if (!inWindow(reading, ['captured_at'], now, windowMs)) continue
    const station = agentById.get(String(reading.agent_id))
    const satellite = satelliteById.get(String(reading.satellite_id))
    if (!station || !satellite) continue
    const key = `${station.id}|${satellite.id}`
    const current = observationPairs.get(key)
    if (current) current.count += 1
    else observationPairs.set(key, { station: station.id, satellite: satellite.id, count: 1 })
  }
  for (const pair of observationPairs.values()) addEdge(pair.station, pair.satellite, 'observed', 0.72, pair.count)

  for (const [index, anomaly] of (input.anomalies ?? []).entries()) {
    if (!inWindow(anomaly, ['detected_at', 'created_at'], now, windowMs)) continue
    const identity = anomaly.id || anomaly.reading_id || `${anomaly.agent_id || 'rf'}-${index}`
    const node = addNode({
      id: idFor('rf', identity),
      kind: 'rf',
      label: `${displayMetric(anomaly.metric)} anomaly`,
      subtitle: anomaly.agent_id ? `${anomaly.agent_id} · ${(Number(anomaly.z_score) || 0).toFixed(1)}σ` : undefined,
      severity: anomalySeverity(anomaly),
      timestamp: timestamp(anomaly, ['detected_at', 'created_at']),
      entityId: String(identity),
      target: 'signal',
      metadata: anomaly,
    })
    evidenceCount += 1
    addEdge(agentById.get(String(anomaly.agent_id))?.id, node.id, 'detected', 0.95)
    addEdge(satelliteById.get(String(anomaly.satellite_id))?.id, node.id, 'affected', 0.95)
  }

  const incidentNodes: CorrelationNode[] = []
  for (const [index, incident] of asItems(input.incidents).entries()) {
    if (!inWindow(incident, ['updated_at', 'created_at'], now, windowMs)) continue
    const identity = incident.id || incident.reference || `incident-${index}`
    const node = addNode({
      id: idFor('incident', identity),
      kind: 'incident',
      label: incident.title || incident.reference || `Incident ${index + 1}`,
      subtitle: incident.reference || incident.status,
      severity: severity(incident.severity),
      timestamp: timestamp(incident, ['updated_at', 'created_at']),
      entityId: String(identity),
      target: 'incidents',
      metadata: incident,
    })
    incidentNodes.push(node)
    evidenceCount += 1
    addEdge(satelliteById.get(String(incident.satellite_id))?.id, node.id, 'affected', 1)
    addEdge(agentById.get(String(incident.agent_id))?.id, node.id, 'affected', 1)
  }

  for (const [index, alert] of asItems(input.alerts).entries()) {
    if (!inWindow(alert, ['created_at', 'updated_at'], now, windowMs)) continue
    const cveId = String(alert.cve_id || '').trim()
    if (!cveId) continue
    const cveMetadata = cveByValue.get(normalized(cveId)) ?? alert
    const cveNode = addNode({
      id: idFor('cve', cveId),
      kind: 'cve',
      label: cveId,
      subtitle: cveMetadata.summary || cveMetadata.description || alert.matched_keyword,
      severity: severity(alert.severity || cveMetadata.severity),
      timestamp: timestamp(alert, ['created_at', 'updated_at']) || timestamp(cveMetadata, ['published_at', 'published']),
      entityId: cveId,
      target: 'intel',
      metadata: { ...cveMetadata, alert },
    })
    evidenceCount += 1
    const assetValue = String(alert.asset_value || '').trim()
    let assetNode = assetByValue.get(normalized(assetValue))
    if (!assetNode && assetValue) {
      assetNode = addNode({
        id: idFor('asset', `exposure-${index}-${assetValue}`),
        kind: 'asset',
        label: assetValue,
        subtitle: 'Exposure match',
        severity: severity(alert.severity),
        timestamp: timestamp(alert, ['created_at', 'updated_at']),
        target: 'assets',
        metadata: { value: assetValue, contextual: true },
      })
      assetByValue.set(normalized(assetValue), assetNode)
      const satellite = satelliteByValue.get(normalized(assetValue))
      if (satellite) addEdge(satellite.id, assetNode.id, 'matched', 0.8)
    }
    addEdge(assetNode?.id, cveNode.id, 'exposed', 1)
    const satellite = satelliteByValue.get(normalized(assetValue))
    if (satellite && !assetNode) addEdge(satellite.id, cveNode.id, 'exposed', 0.95)
  }

  const cveNodes = [...nodes.values()].filter((node) => node.kind === 'cve')
  for (const incident of incidentNodes) {
    const haystack = normalized(`${incident.label} ${incident.subtitle || ''} ${incident.metadata?.description || ''}`)
    for (const cve of cveNodes) {
      if (haystack.includes(normalized(cve.label))) addEdge(cve.id, incident.id, 'mentioned', 0.9)
    }
  }

  const nodeList = [...nodes.values()]
  return {
    nodes: nodeList,
    edges: [...edges.values()],
    evidenceCount,
    staticCount: Math.max(0, nodeList.length - evidenceCount),
  }
}

export function graphAdjacency(graph: CorrelationGraph): Map<string, Set<string>> {
  const adjacency = new Map<string, Set<string>>()
  for (const node of graph.nodes) adjacency.set(node.id, new Set())
  for (const edge of graph.edges) {
    adjacency.get(edge.source)?.add(edge.target)
    adjacency.get(edge.target)?.add(edge.source)
  }
  return adjacency
}

export function graphDegrees(graph: CorrelationGraph): Map<string, number> {
  const result = new Map<string, number>()
  for (const node of graph.nodes) result.set(node.id, 0)
  for (const edge of graph.edges) {
    result.set(edge.source, (result.get(edge.source) ?? 0) + edge.count)
    result.set(edge.target, (result.get(edge.target) ?? 0) + edge.count)
  }
  return result
}

export function chooseCorrelationAnchor(graph: CorrelationGraph): string | null {
  if (!graph.nodes.length) return null
  const degrees = graphDegrees(graph)
  const ranked = [...graph.nodes].sort((a, b) => {
    const kindRank = (node: CorrelationNode) => node.kind === 'satellite' ? 3 : node.kind === 'station' ? 2 : node.kind === 'agent' ? 1 : 0
    return kindRank(b) - kindRank(a) || (degrees.get(b.id) ?? 0) - (degrees.get(a.id) ?? 0) || a.label.localeCompare(b.label)
  })
  return ranked[0]?.id ?? null
}

export function correlationDistances(graph: CorrelationGraph, sourceId: string): Map<string, number> {
  const adjacency = graphAdjacency(graph)
  const distances = new Map<string, number>([[sourceId, 0]])
  const queue = [sourceId]
  while (queue.length) {
    const current = queue.shift() as string
    const distance = distances.get(current) ?? 0
    for (const next of adjacency.get(current) ?? []) {
      if (distances.has(next)) continue
      distances.set(next, distance + 1)
      queue.push(next)
    }
  }
  return distances
}

export function correlationNeighborhood(graph: CorrelationGraph, sourceId: string, depth: number, maxNodes = 46): Set<string> {
  const distances = correlationDistances(graph, sourceId)
  const degrees = graphDegrees(graph)
  const ranked = graph.nodes
    .filter((node) => (distances.get(node.id) ?? Number.POSITIVE_INFINITY) <= depth)
    .sort((a, b) => (distances.get(a.id) ?? 99) - (distances.get(b.id) ?? 99) || (degrees.get(b.id) ?? 0) - (degrees.get(a.id) ?? 0) || a.label.localeCompare(b.label))
    .slice(0, maxNodes)
  return new Set(ranked.map((node) => node.id))
}

export function shortestCorrelationPath(graph: CorrelationGraph, sourceId: string, targetId: string): string[] {
  if (sourceId === targetId) return graph.nodes.some((node) => node.id === sourceId) ? [sourceId] : []
  const adjacency = graphAdjacency(graph)
  const previous = new Map<string, string | null>([[sourceId, null]])
  const queue = [sourceId]
  while (queue.length) {
    const current = queue.shift() as string
    for (const next of adjacency.get(current) ?? []) {
      if (previous.has(next)) continue
      previous.set(next, current)
      if (next === targetId) {
        const path = [targetId]
        let cursor: string | null = current
        while (cursor) {
          path.push(cursor)
          cursor = previous.get(cursor) ?? null
        }
        return path.reverse()
      }
      queue.push(next)
    }
  }
  return []
}

export function orbitLayout3D(
  graph: CorrelationGraph,
  focusId: string,
  visibleIds: Set<string>,
  camera: CorrelationCamera = { yaw: -0.38, pitch: -0.16 },
  width = 1120,
  height = 680,
): CorrelationPoint[] {
  const distances = correlationDistances(graph, focusId)
  const centerX = width / 2
  const centerY = height / 2
  const rings = [0, 150, 265, 360]
  const groups = new Map<number, CorrelationNode[]>()
  for (const node of graph.nodes) {
    if (!visibleIds.has(node.id)) continue
    const distance = Math.min(3, distances.get(node.id) ?? 3)
    const group = groups.get(distance) ?? []
    group.push(node)
    groups.set(distance, group)
  }

  const cosYaw = Math.cos(camera.yaw)
  const sinYaw = Math.sin(camera.yaw)
  const cosPitch = Math.cos(camera.pitch)
  const sinPitch = Math.sin(camera.pitch)
  const cameraDistance = 980
  const points: CorrelationPoint[] = []

  for (const [distance, group] of [...groups.entries()].sort((a, b) => a[0] - b[0])) {
    const sorted = group.sort((a, b) => a.kind.localeCompare(b.kind) || a.label.localeCompare(b.label))
    if (distance === 0) {
      points.push({ node: sorted[0], x: centerX, y: centerY, z: 0, scale: 1, distance })
      continue
    }

    const radius = rings[distance] ?? rings[3]
    const offset = distance % 2 === 0 ? Math.PI / Math.max(2, sorted.length) : -Math.PI / 2
    const planeTilt = distance === 1 ? -0.16 : distance === 2 ? 0.1 : -0.07
    sorted.forEach((node, index) => {
      const angle = offset + (Math.PI * 2 * index) / Math.max(1, sorted.length)
      const worldX = Math.cos(angle) * radius
      const worldZ = Math.sin(angle) * radius
      const worldY = Math.sin(angle + distance * 0.78) * radius * planeTilt
      const rotatedX = worldX * cosYaw - worldZ * sinYaw
      const yawZ = worldX * sinYaw + worldZ * cosYaw
      const rotatedY = worldY * cosPitch - yawZ * sinPitch
      const rotatedZ = worldY * sinPitch + yawZ * cosPitch
      const perspective = Math.max(0.72, Math.min(1.28, cameraDistance / (cameraDistance + rotatedZ)))
      points.push({
        node,
        x: centerX + rotatedX * perspective,
        y: centerY + rotatedY * perspective,
        z: rotatedZ,
        scale: perspective,
        distance,
      })
    })
  }

  return points.sort((a, b) => b.z - a.z || b.distance - a.distance)
}

export function projectOrbitRing3D(
  radius: number,
  camera: CorrelationCamera,
  distance: number,
  width = 1120,
  height = 680,
): string {
  const centerX = width / 2
  const centerY = height / 2
  const planeTilt = distance === 1 ? -0.16 : distance === 2 ? 0.1 : -0.07
  const cosYaw = Math.cos(camera.yaw)
  const sinYaw = Math.sin(camera.yaw)
  const cosPitch = Math.cos(camera.pitch)
  const sinPitch = Math.sin(camera.pitch)
  const cameraDistance = 980
  const points: string[] = []
  for (let step = 0; step <= 64; step += 1) {
    const angle = (Math.PI * 2 * step) / 64
    const worldX = Math.cos(angle) * radius
    const worldZ = Math.sin(angle) * radius
    const worldY = Math.sin(angle + distance * 0.78) * radius * planeTilt
    const rotatedX = worldX * cosYaw - worldZ * sinYaw
    const yawZ = worldX * sinYaw + worldZ * cosYaw
    const rotatedY = worldY * cosPitch - yawZ * sinPitch
    const rotatedZ = worldY * sinPitch + yawZ * cosPitch
    const perspective = Math.max(0.72, Math.min(1.28, cameraDistance / (cameraDistance + rotatedZ)))
    const x = centerX + rotatedX * perspective
    const y = centerY + rotatedY * perspective
    points.push(`${step === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`)
  }
  return points.join(' ')
}

export function orbitLayout(graph: CorrelationGraph, focusId: string, visibleIds: Set<string>, width = 1120, height = 680): CorrelationPoint[] {
  return orbitLayout3D(graph, focusId, visibleIds, { yaw: 0, pitch: 0 }, width, height)
}


