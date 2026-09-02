import { useEffect, useMemo, useRef, useState, type KeyboardEvent, type PointerEvent, type ReactNode } from 'react'
import { Activity, ArrowUpRight, Boxes, Crosshair, Orbit, Radar, RadioTower, RotateCcw, Search, ShieldAlert, ZoomIn, ZoomOut } from 'lucide-react'
import { useI18n } from '../lib/I18nContext'
import type { RouteTab } from '../lib/routeTabs'
import {
  buildCorrelationGraph,
  chooseCorrelationAnchor,
  correlationNeighborhood,
  graphDegrees,
  orbitLayout3D,
  projectOrbitRing3D,
  shortestCorrelationPath,
  type CorrelationCamera,
  type CorrelationEdge,
  type CorrelationGraph as CorrelationGraphModel,
  type CorrelationNode,
  type CorrelationNodeKind,
  type CorrelationRecord,
  type CorrelationSeverity,
} from '../lib/correlationGraph'
import StatusState from './StatusState'

const DAY_MS = 24 * 60 * 60 * 1000
const KIND_ORDER: CorrelationNodeKind[] = ['satellite', 'station', 'agent', 'incident', 'rf', 'asset', 'cve', 'program']
const SEVERITY_RANK: Record<CorrelationSeverity, number> = { unknown: 0, low: 1, medium: 2, high: 3, critical: 4 }

type WindowKey = '24h' | '7d' | '30d' | 'all'
type SeverityFilter = 'all' | 'high' | 'critical'

interface CorrelationGraphProps {
  satellites?: CorrelationRecord[] | null
  agents?: CorrelationRecord[] | null
  readings?: CorrelationRecord[] | null
  anomalies?: CorrelationRecord[] | null
  incidents?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  assets?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  programs?: CorrelationRecord[] | null
  alerts?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  cves?: CorrelationRecord[] | { items?: CorrelationRecord[] } | null
  access?: Record<string, boolean>
  onNavigate: (tab: RouteTab) => void
  onOpenSatellite?: (id: string) => void
  now?: number
}

function windowMs(value: WindowKey): number | null {
  if (value === '24h') return DAY_MS
  if (value === '7d') return 7 * DAY_MS
  if (value === '30d') return 30 * DAY_MS
  return null
}

function nodeTone(kind: CorrelationNodeKind): string {
  if (kind === 'incident') return 'rgb(var(--lyra-bad))'
  if (kind === 'rf') return 'rgb(var(--lyra-warn))'
  if (kind === 'station' || kind === 'agent') return 'rgb(var(--lyra-good))'
  if (kind === 'cve') return 'rgb(var(--lyra-gold-bright))'
  if (kind === 'asset' || kind === 'program') return 'rgb(var(--lyra-muted))'
  return 'rgb(var(--lyra-gold))'
}

function severityTone(severity: CorrelationSeverity): string {
  if (severity === 'critical' || severity === 'high') return 'rgb(var(--lyra-bad))'
  if (severity === 'medium') return 'rgb(var(--lyra-warn))'
  if (severity === 'low') return 'rgb(var(--lyra-good))'
  return 'rgb(var(--lyra-border))'
}

function nodeRadius(node: CorrelationNode, selected: boolean): number {
  if (selected) return 22
  if (node.kind === 'satellite') return 18
  if (node.kind === 'incident' || node.kind === 'cve') return 16
  return 14
}

function truncate(value: string, length = 22): string {
  return value.length > length ? `${value.slice(0, length - 1)}…` : value
}

function capitalize(value: string): string {
  return value ? value.charAt(0).toLocaleUpperCase() + value.slice(1) : value
}

function nodeIcon(kind: CorrelationNodeKind) {
  if (kind === 'satellite') return Orbit
  if (kind === 'station' || kind === 'agent') return RadioTower
  if (kind === 'incident') return ShieldAlert
  if (kind === 'rf') return Activity
  if (kind === 'asset' || kind === 'program') return Boxes
  return Radar
}

function severityMatches(node: CorrelationNode, filter: SeverityFilter): boolean {
  if (filter === 'all' || node.severity === 'unknown') return true
  return SEVERITY_RANK[node.severity] >= SEVERITY_RANK[filter]
}

function edgeInPath(edge: CorrelationEdge, path: string[]): boolean {
  for (let index = 0; index < path.length - 1; index += 1) {
    const a = path[index]
    const b = path[index + 1]
    if ((edge.source === a && edge.target === b) || (edge.source === b && edge.target === a)) return true
  }
  return false
}

function nodeDescription(node: CorrelationNode): string {
  const values = [node.subtitle, node.metadata?.status, node.metadata?.kind, node.metadata?.metric]
  return values.find((value) => String(value ?? '').trim()) || '—'
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const parsed = Date.parse(value)
  return Number.isFinite(parsed) ? new Date(parsed).toLocaleString() : '—'
}

function relationKey(relation: CorrelationEdge['relation']): string {
  return `correlationGraph.relations.${relation}`
}

function renderNodeShape(node: CorrelationNode, radius: number, tone: string) {
  if (node.kind === 'incident') {
    return <path d={`M 0 ${-radius} L ${radius * 0.9} ${radius * 0.78} L ${-radius * 0.9} ${radius * 0.78} Z`} fill="rgb(var(--lyra-panel))" stroke={tone} strokeWidth="2" />
  }
  if (node.kind === 'station' || node.kind === 'agent') {
    return <path d={`M 0 ${-radius} L ${radius} 0 L 0 ${radius} L ${-radius} 0 Z`} fill="rgb(var(--lyra-panel))" stroke={tone} strokeWidth="2" />
  }
  if (node.kind === 'asset' || node.kind === 'program') {
    return <rect x={-radius} y={-radius} width={radius * 2} height={radius * 2} rx={node.kind === 'program' ? 7 : 3} fill="rgb(var(--lyra-panel))" stroke={tone} strokeWidth="2" />
  }
  if (node.kind === 'cve') {
    const d = radius * 0.72
    return <path d={`M ${-d} ${-radius} L ${d} ${-radius} L ${radius} ${-d} L ${radius} ${d} L ${d} ${radius} L ${-d} ${radius} L ${-radius} ${d} L ${-radius} ${-d} Z`} fill="rgb(var(--lyra-panel))" stroke={tone} strokeWidth="2" />
  }
  return <circle r={radius} fill="rgb(var(--lyra-panel))" stroke={tone} strokeWidth="2" />
}

function GraphNodeGlyph({ node, selected, pathActive }: { node: CorrelationNode; selected: boolean; pathActive: boolean }) {
  const tone = nodeTone(node.kind)
  const radius = nodeRadius(node, selected)
  return (
    <>
      {(selected || pathActive) && <circle r={radius + 7} fill="none" stroke={tone} strokeWidth="1" opacity={selected ? 0.52 : 0.3} />}
      {renderNodeShape(node, radius, tone)}
      <circle r="2.7" fill={tone} />
    </>
  )
}

export default function CorrelationGraph({ satellites, agents, readings, anomalies, incidents, assets, programs, alerts, cves, access = {}, onNavigate, onOpenSatellite, now }: CorrelationGraphProps) {
  const { t } = useI18n()
  const [mountedAt] = useState(Date.now)
  const graphNow = now ?? mountedAt
  const [timeWindow, setTimeWindow] = useState<WindowKey>('7d')
  const [depth, setDepth] = useState(2)
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all')
  const [enabledKinds, setEnabledKinds] = useState<Set<CorrelationNodeKind>>(() => new Set(KIND_ORDER))
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [routeOriginId, setRouteOriginId] = useState<string | null>(null)
  const [routeTargetId, setRouteTargetId] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [zoom, setZoom] = useState(1)
  const [camera, setCamera] = useState<CorrelationCamera>({ yaw: -0.38, pitch: -0.16 })
  const dragRef = useRef<{ x: number; y: number; yaw: number; pitch: number } | null>(null)

  const graph = useMemo(() => buildCorrelationGraph({ satellites, agents, readings, anomalies, incidents, assets, programs, alerts, cves, now: graphNow, windowMs: windowMs(timeWindow) }), [satellites, agents, readings, anomalies, incidents, assets, programs, alerts, cves, graphNow, timeWindow])
  const nodeMap = useMemo(() => new Map(graph.nodes.map((node) => [node.id, node])), [graph.nodes])
  const degrees = useMemo(() => graphDegrees(graph), [graph])

  useEffect(() => {
    if (selectedId && nodeMap.has(selectedId)) return
    setSelectedId(chooseCorrelationAnchor(graph))
    setRouteOriginId(null)
    setRouteTargetId(null)
  }, [graph, nodeMap, selectedId])

  const selected = selectedId ? nodeMap.get(selectedId) ?? null : null
  const path = useMemo(() => routeOriginId && routeTargetId ? shortestCorrelationPath(graph, routeOriginId, routeTargetId) : [], [graph, routeOriginId, routeTargetId])
  const pathIds = useMemo(() => new Set(path), [path])

  const filteredGraph = useMemo<CorrelationGraphModel>(() => {
    const nodes = graph.nodes.filter((node) => enabledKinds.has(node.kind) && severityMatches(node, severityFilter))
    if (selected && !nodes.some((node) => node.id === selected.id)) nodes.push(selected)
    const ids = new Set(nodes.map((node) => node.id))
    return { ...graph, nodes, edges: graph.edges.filter((edge) => ids.has(edge.source) && ids.has(edge.target)) }
  }, [graph, enabledKinds, severityFilter, selected])

  const visibleIds = useMemo(() => {
    if (!selectedId) return new Set<string>()
    const result = correlationNeighborhood(filteredGraph, selectedId, depth)
    const contextTarget = Math.min(30, filteredGraph.nodes.length)
    if (result.size < Math.min(12, contextTarget)) {
      const rankedContext = [...filteredGraph.nodes].sort((a, b) => (degrees.get(b.id) ?? 0) - (degrees.get(a.id) ?? 0) || a.label.localeCompare(b.label))
      for (const node of rankedContext) {
        result.add(node.id)
        if (result.size >= contextTarget) break
      }
    }
    for (const id of path) result.add(id)
    return result
  }, [filteredGraph, selectedId, depth, path, degrees])

  const points = useMemo(() => selectedId ? orbitLayout3D(graph, selectedId, visibleIds, camera) : [], [graph, selectedId, visibleIds, camera])
  const pointMap = useMemo(() => new Map(points.map((point) => [point.node.id, point])), [points])
  const visibleEdges = useMemo(() => graph.edges.filter((edge) => visibleIds.has(edge.source) && visibleIds.has(edge.target)), [graph.edges, visibleIds])
  const visibleNodes = points.map((point) => point.node)

  const searchMatches = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase()
    if (!needle) return []
    return graph.nodes
      .filter((node) => `${node.label} ${node.subtitle || ''} ${node.kind}`.toLocaleLowerCase().includes(needle))
      .sort((a, b) => (degrees.get(b.id) ?? 0) - (degrees.get(a.id) ?? 0) || a.label.localeCompare(b.label))
      .slice(0, 8)
  }, [query, graph.nodes, degrees])

  const related = useMemo(() => {
    if (!selected) return []
    return graph.edges
      .filter((edge) => edge.source === selected.id || edge.target === selected.id)
      .map((edge) => {
        const otherId = edge.source === selected.id ? edge.target : edge.source
        return { edge, node: nodeMap.get(otherId) }
      })
      .filter((item): item is { edge: CorrelationEdge; node: CorrelationNode } => Boolean(item.node))
      .sort((a, b) => b.edge.strength - a.edge.strength || b.edge.count - a.edge.count)
  }, [graph.edges, selected, nodeMap])

  const sourceNode = routeOriginId ? nodeMap.get(routeOriginId) ?? null : null
  const targetNode = routeTargetId ? nodeMap.get(routeTargetId) ?? null : null
  const activeSourceCount = Object.values(access).filter(Boolean).length

  function selectNode(id: string) {
    setSelectedId(id)
    if (routeOriginId && id !== routeOriginId) setRouteTargetId(id)
    const node = nodeMap.get(id)
    if (node && !enabledKinds.has(node.kind)) setEnabledKinds((current) => new Set([...current, node.kind]))
  }

  function handleNodeKey(event: KeyboardEvent<SVGGElement>, id: string) {
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    selectNode(id)
  }

  function toggleKind(kind: CorrelationNodeKind) {
    setEnabledKinds((current) => {
      const next = new Set(current)
      if (next.has(kind)) next.delete(kind)
      else next.add(kind)
      return next
    })
  }

  function startRoute() {
    if (!selected) return
    setRouteOriginId(selected.id)
    setRouteTargetId(null)
  }

  function clearRoute() {
    setRouteOriginId(null)
    setRouteTargetId(null)
  }

  function resetView() {
    setDepth(2)
    setSeverityFilter('all')
    setEnabledKinds(new Set(KIND_ORDER))
    setQuery('')
    setZoom(1)
    setCamera({ yaw: -0.38, pitch: -0.16 })
    setRouteOriginId(null)
    setRouteTargetId(null)
    setSelectedId(chooseCorrelationAnchor(graph))
  }


  function handleScenePointerDown(event: PointerEvent<SVGSVGElement>) {
    if (event.button !== 0) return
    dragRef.current = { x: event.clientX, y: event.clientY, yaw: camera.yaw, pitch: camera.pitch }
    event.currentTarget.setPointerCapture(event.pointerId)
  }

  function handleScenePointerMove(event: PointerEvent<SVGSVGElement>) {
    const drag = dragRef.current
    if (!drag) return
    const yaw = drag.yaw + (event.clientX - drag.x) * 0.006
    const pitch = Math.max(-0.62, Math.min(0.42, drag.pitch + (event.clientY - drag.y) * 0.0045))
    setCamera({ yaw, pitch })
  }

  function handleScenePointerUp(event: PointerEvent<SVGSVGElement>) {
    dragRef.current = null
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId)
  }

  function edgeCurve(source: { x: number; y: number; z: number }, target: { x: number; y: number; z: number }): string {
    const midX = (source.x + target.x) / 2
    const midY = (source.y + target.y) / 2 - Math.min(26, Math.abs(source.z - target.z) * 0.05 + 8)
    return `M ${source.x.toFixed(2)} ${source.y.toFixed(2)} Q ${midX.toFixed(2)} ${midY.toFixed(2)} ${target.x.toFixed(2)} ${target.y.toFixed(2)}`
  }

  function openSelected() {
    if (!selected) return
    if (selected.kind === 'satellite' && selected.entityId && onOpenSatellite) {
      onOpenSatellite(selected.entityId)
      return
    }
    onNavigate(selected.target)
  }

  if (!graph.nodes.length) {
    return <StatusState kind="empty" title={t('correlationGraph.empty')} message={t('correlationGraph.emptyHint')} />
  }

  return (
    <section aria-labelledby="correlation-graph-heading" className="relative">
      <div className="relative min-h-[760px] overflow-hidden rounded-2xl border border-lyra-borderSoft bg-[#06080b] xl:min-h-[calc(100vh-118px)]">
        <div className="pointer-events-none absolute inset-0 lyra-correlation-stage" />

        <div className="pointer-events-none absolute left-5 top-5 z-20 max-w-xl">
          <div className="font-mono text-[9px] tracking-[0.22em] text-lyra-mutedDim">{t('correlationGraph.eyebrow')}</div>
          <h1 id="correlation-graph-heading" className="mt-1.5 font-display text-xl font-medium tracking-tight text-lyra-text sm:text-2xl">{t('correlationGraph.title')}</h1>
          <p className="mt-1.5 hidden max-w-lg text-xs leading-5 text-lyra-muted lg:block">{t('correlationGraph.subtitle')}</p>
        </div>

        <div className="absolute right-4 top-4 z-30 flex items-center gap-1 rounded-xl border border-lyra-borderSoft bg-[#0a0d11]/88 p-1.5 shadow-2xl shadow-black/30 backdrop-blur-md">
          <IconButton label={t('correlationGraph.zoomIn')} onClick={() => setZoom((value) => Math.min(1.35, Number((value + 0.1).toFixed(2))))}><ZoomIn className="h-4 w-4" /></IconButton>
          <IconButton label={t('correlationGraph.zoomOut')} onClick={() => setZoom((value) => Math.max(0.75, Number((value - 0.1).toFixed(2))))}><ZoomOut className="h-4 w-4" /></IconButton>
          <IconButton label={t('correlationGraph.recenter')} onClick={() => { setZoom(1); setCamera({ yaw: -0.38, pitch: -0.16 }) }}><Crosshair className="h-4 w-4" /></IconButton>
          <span className="mx-1 h-5 w-px bg-lyra-borderSoft" />
          <IconButton label={t('correlationGraph.reset')} onClick={resetView}><RotateCcw className="h-4 w-4" /></IconButton>
        </div>

        <div className="absolute left-4 top-[108px] z-20 hidden items-center gap-2 rounded-xl border border-lyra-borderSoft bg-[#0a0d11]/82 p-1.5 backdrop-blur-md lg:flex">
          {(['24h', '7d', '30d', 'all'] as WindowKey[]).map((value) => (
            <FilterButton key={value} active={timeWindow === value} onClick={() => setTimeWindow(value)} label={t(`correlationGraph.window.${value}`)} />
          ))}
          <span className="mx-1 h-5 w-px bg-lyra-borderSoft" />
          <ControlGroup label={t('correlationGraph.depth')}>
            {[1, 2, 3].map((value) => <FilterButton key={value} active={depth === value} onClick={() => setDepth(value)} label={`${value}`} />)}
          </ControlGroup>
        </div>

        <div className="absolute bottom-4 left-4 top-[158px] z-10 hidden w-[156px] content-start gap-1.5 xl:grid">
          {KIND_ORDER.map((kind) => (
            <button key={kind} type="button" aria-pressed={enabledKinds.has(kind)} onClick={() => toggleKind(kind)} className={`flex items-center justify-between gap-2 rounded-lg border px-2.5 py-2 text-left text-[10px] transition ${enabledKinds.has(kind) ? 'border-lyra-border bg-[#0b0e12]/82 text-lyra-text' : 'border-transparent bg-transparent text-lyra-mutedDim opacity-55 hover:opacity-90'}`}>
              <span className="inline-flex items-center gap-2"><span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: nodeTone(kind) }} />{t(`correlationGraph.kind.${kind}`)}</span>
            </button>
          ))}
        </div>

        <div className="absolute right-4 top-[72px] z-20 hidden grid-cols-3 gap-1 rounded-xl border border-lyra-borderSoft bg-[#0a0d11]/82 p-1.5 backdrop-blur-md sm:grid">
          <Metric label={t('correlationGraph.metrics.nodes')} value={graph.nodes.length} />
          <Metric label={t('correlationGraph.metrics.links')} value={graph.edges.length} />
          <Metric label={t('correlationGraph.metrics.evidence')} value={graph.evidenceCount} />
        </div>

        <div className="absolute inset-0 xl:left-[164px] xl:right-[344px]">
          <svg
            viewBox="0 0 1120 680"
            role="group"
            aria-label={t('correlationGraph.canvasLabel')}
            className="absolute inset-0 h-full w-full cursor-grab touch-none active:cursor-grabbing"
            onPointerDown={handleScenePointerDown}
            onPointerMove={handleScenePointerMove}
            onPointerUp={handleScenePointerUp}
            onPointerCancel={handleScenePointerUp}
          >
            <defs>
              <filter id="lyra-node-shadow" x="-60%" y="-60%" width="220%" height="220%">
                <feDropShadow dx="0" dy="5" stdDeviation="6" floodColor="#000" floodOpacity="0.42" />
              </filter>
            </defs>
            <g transform={`translate(560 340) scale(${zoom}) translate(-560 -340)`}>
              {[150, 265, 360].map((radius, index) => (
                <path key={radius} d={projectOrbitRing3D(radius, camera, index + 1)} fill="none" stroke="rgb(var(--lyra-border))" strokeOpacity={index === 0 ? 0.26 : 0.13} strokeWidth="1" />
              ))}

              {visibleEdges.map((edge) => {
                const source = pointMap.get(edge.source)
                const target = pointMap.get(edge.target)
                if (!source || !target) return null
                const active = edgeInPath(edge, path)
                const selectedEdge = selected ? edge.source === selected.id || edge.target === selected.id : false
                return (
                  <path key={edge.id} d={edgeCurve(source, target)} fill="none" stroke={active ? 'rgb(var(--lyra-gold-bright))' : selectedEdge ? 'rgb(var(--lyra-gold))' : 'rgb(var(--lyra-border))'} strokeWidth={active ? 2 : selectedEdge ? 1.15 : 0.75} strokeOpacity={active ? 0.94 : selectedEdge ? 0.52 : 0.2} strokeDasharray={active ? '6 8' : undefined} className={active ? 'lyra-correlation-flow' : undefined} pointerEvents="none">
                    <title>{t(relationKey(edge.relation))}{edge.count > 1 ? ` ×${edge.count}` : ''}</title>
                  </path>
                )
              })}

              {points.map((point) => {
                const node = point.node
                const isSelected = node.id === selectedId
                const pathActive = pathIds.has(node.id)
                const faded = path.length > 1 && !pathActive
                const depthOpacity = Math.max(0.38, Math.min(1, 0.72 + point.scale * 0.2))
                const scale = (isSelected ? 1.12 : 1) * point.scale
                const radius = nodeRadius(node, isSelected)
                return (
                  <g key={node.id} transform={`translate(${point.x} ${point.y}) scale(${scale})`} role="button" tabIndex={0} aria-label={`${t(`correlationGraph.kind.${node.kind}`)}: ${node.label}`} aria-pressed={isSelected} onClick={(event) => { event.stopPropagation(); selectNode(node.id) }} onKeyDown={(event) => handleNodeKey(event, node.id)} className="cursor-pointer outline-none transition-opacity focus-visible:opacity-100" opacity={faded ? 0.16 : depthOpacity} filter="url(#lyra-node-shadow)">
                    <GraphNodeGlyph node={node} selected={isSelected} pathActive={pathActive} />
                    <text y={radius + 18} textAnchor="middle" fill={isSelected ? 'rgb(var(--lyra-gold-bright))' : 'rgb(var(--lyra-text))'} fontSize={isSelected ? 11 : 9.2} fontWeight={isSelected ? 600 : 430}>{truncate(node.label, isSelected ? 28 : 17)}</text>
                    {isSelected && <text y={radius + 31} textAnchor="middle" fill="rgb(var(--lyra-muted-dim))" fontSize="7.5" letterSpacing="1.2">{t(`correlationGraph.kind.${node.kind}`)}</text>}
                  </g>
                )
              })}
            </g>
          </svg>

          {graph.edges.length === 0 && (
            <div className="pointer-events-none absolute left-1/2 top-[74%] -translate-x-1/2 rounded-full border border-lyra-borderSoft bg-[#090c10]/74 px-3 py-1.5 text-[10px] text-lyra-mutedDim backdrop-blur">
              {t('correlationGraph.awaitingEvidence', { count: graph.nodes.length })}
            </div>
          )}
        </div>

        <aside aria-label={t('correlationGraph.contextPanel')} className="absolute bottom-4 right-4 top-[128px] z-20 hidden w-[324px] overflow-y-auto rounded-2xl border border-lyra-borderSoft bg-[#0a0d11]/90 p-4 shadow-2xl shadow-black/30 backdrop-blur-md xl:block">
          {selected ? (
            <>
              <div>
                <div className="flex items-start justify-between gap-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-lyra-borderSoft bg-lyra-bgSoft/40" style={{ color: nodeTone(selected.kind) }}>
                      {(() => { const Icon = nodeIcon(selected.kind); return <Icon className="h-[18px] w-[18px]" strokeWidth={1.6} /> })()}
                    </span>
                    <div className="min-w-0">
                      <p className="font-mono text-[8px] tracking-[0.16em] text-lyra-mutedDim">{t(`correlationGraph.kind.${selected.kind}`)}</p>
                      <h2 className="mt-1 break-words text-sm font-medium text-lyra-text">{selected.label}</h2>
                    </div>
                  </div>
                  {selected.severity !== 'unknown' && <span className="rounded-full border px-2 py-1 font-mono text-[8px]" style={{ borderColor: severityTone(selected.severity), color: severityTone(selected.severity) }}>{capitalize(selected.severity)}</span>}
                </div>
                <p className="mt-3 text-[11px] leading-5 text-lyra-muted">{nodeDescription(selected)}</p>
              </div>

              <div className="mt-4 grid grid-cols-2 gap-2">
                <Detail label={t('correlationGraph.connections')} value={degrees.get(selected.id) ?? 0} />
                <Detail label={t('correlationGraph.lastEvidence')} value={formatTime(selected.timestamp)} />
              </div>

              <div className="mt-3 grid grid-cols-2 gap-2">
                <button type="button" onClick={openSelected} className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-lyra-gold/30 bg-lyra-gold/10 px-2.5 py-2 text-[10px] font-medium text-lyra-goldBright transition hover:bg-lyra-gold/15">{t('correlationGraph.openWorkspace')} <ArrowUpRight className="h-3 w-3" aria-hidden="true" /></button>
                <button type="button" onClick={startRoute} className="inline-flex items-center justify-center gap-1.5 rounded-lg border border-lyra-border px-2.5 py-2 text-[10px] text-lyra-muted transition hover:border-lyra-gold/35 hover:text-lyra-text"><Crosshair className="h-3 w-3" aria-hidden="true" /> {t('correlationGraph.traceFromHere')}</button>
              </div>

              {(sourceNode || targetNode) && (
                <div className="mt-4 rounded-xl border border-lyra-gold/18 bg-lyra-gold/5 p-3">
                  <div className="flex items-center justify-between gap-2"><p className="font-mono text-[8px] tracking-[0.14em] text-lyra-gold">{t('correlationGraph.route.title')}</p><button type="button" onClick={clearRoute} className="text-[9px] text-lyra-muted hover:text-lyra-text">{t('correlationGraph.route.clear')}</button></div>
                  <p className="mt-2 text-[10px] text-lyra-muted">{sourceNode?.label || '—'} <span aria-hidden="true">→</span> {targetNode?.label || t('correlationGraph.route.waiting')}</p>
                  {targetNode && !path.length && <p role="status" className="mt-2 text-[10px] text-lyra-warn">{t('correlationGraph.route.noPath')}</p>}
                  {path.length > 1 && <ol className="mt-3 grid gap-1.5">{path.map((id, index) => { const node = nodeMap.get(id); if (!node) return null; return <li key={id} className="flex items-center gap-2 text-[10px] text-lyra-text"><span className="font-mono text-lyra-gold">{String(index + 1).padStart(2, '0')}</span><span className="truncate">{node.label}</span></li> })}</ol>}
                </div>
              )}

              <div className="mt-4">
                <div className="mb-2 flex items-center justify-between gap-2"><p className="font-mono text-[8px] tracking-[0.14em] text-lyra-mutedDim">{t('correlationGraph.whyConnected')}</p><span className="text-[9px] text-lyra-mutedDim">{related.length}</span></div>
                <div className="grid gap-1.5">{related.length ? related.slice(0, 8).map(({ edge, node }) => <button key={edge.id} type="button" onClick={() => selectNode(node.id)} className="group rounded-lg border border-lyra-borderSoft bg-lyra-bgSoft/20 p-2 text-left transition hover:border-lyra-gold/25"><span className="flex items-center justify-between gap-2"><span className="truncate text-[10px] text-lyra-text group-hover:text-lyra-goldBright">{node.label}</span><span className="font-mono text-[8px] text-lyra-mutedDim">{edge.count > 1 ? `×${edge.count}` : ''}</span></span><span className="mt-0.5 block text-[9px] text-lyra-mutedDim">{t(relationKey(edge.relation))}</span></button>) : <p className="text-[10px] text-lyra-mutedDim">{t('correlationGraph.noRelations')}</p>}</div>
              </div>
            </>
          ) : <StatusState kind="empty" title={t('correlationGraph.noSelection')} message={t('correlationGraph.interactionHint')} />}
        </aside>

        <div className="absolute bottom-4 left-4 z-30 flex max-w-[calc(100%-32px)] items-end gap-2 xl:left-[180px] xl:right-[360px]">
          <div className="relative min-w-0 flex-1">
            <label className="relative block">
              <span className="sr-only">{t('correlationGraph.search')}</span>
              <Search className="pointer-events-none absolute left-3 top-2.5 h-4 w-4 text-lyra-mutedDim" aria-hidden="true" />
              <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t('correlationGraph.search')} className="w-full rounded-xl border border-lyra-borderSoft bg-[#0a0d11]/92 py-2.5 pl-9 pr-3 text-xs text-lyra-text shadow-2xl shadow-black/20 backdrop-blur-md placeholder:text-lyra-mutedDim" />
            </label>
            {!!query.trim() && <div className="absolute bottom-[48px] left-0 right-0 grid max-h-56 gap-1 overflow-y-auto rounded-xl border border-lyra-borderSoft bg-[#0a0d11]/96 p-2 shadow-2xl shadow-black/40 backdrop-blur-md">{searchMatches.length ? searchMatches.map((node) => <button key={node.id} type="button" onClick={() => { selectNode(node.id); setQuery('') }} className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-left text-[10px] text-lyra-muted transition hover:bg-lyra-gold/10 hover:text-lyra-text"><span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: nodeTone(node.kind) }} />{node.label}</button>) : <span className="px-2 py-2 text-[10px] text-lyra-mutedDim">{t('correlationGraph.noSearchResults')}</span>}</div>}
          </div>
          <div className="hidden rounded-xl border border-lyra-borderSoft bg-[#0a0d11]/88 px-3 py-2.5 text-[9px] text-lyra-mutedDim backdrop-blur-md lg:block">{routeOriginId && !routeTargetId ? t('correlationGraph.selectDestination', { source: sourceNode?.label || '—' }) : t('correlationGraph.interactionHint')}</div>
        </div>
      </div>

      <div className="mt-3 grid gap-3 xl:hidden">
        <div className="flex flex-wrap gap-2">{KIND_ORDER.map((kind) => <FilterButton key={kind} active={enabledKinds.has(kind)} onClick={() => toggleKind(kind)} label={t(`correlationGraph.kind.${kind}`)} tone={nodeTone(kind)} />)}</div>
        <div className="flex flex-wrap gap-2">{(['24h', '7d', '30d', 'all'] as WindowKey[]).map((value) => <FilterButton key={value} active={timeWindow === value} onClick={() => setTimeWindow(value)} label={t(`correlationGraph.window.${value}`)} />)}{(['all', 'high', 'critical'] as SeverityFilter[]).map((value) => <FilterButton key={value} active={severityFilter === value} onClick={() => setSeverityFilter(value)} label={t(`correlationGraph.severityFilter.${value}`)} />)}</div>
      </div>
    </section>
  )
}

function Metric({ label, value }: { label: string; value: number }) {
  return <div className="border-l border-lyra-borderSoft px-3 text-right first:border-l-0"><p className="font-mono text-lg text-lyra-text">{value}</p><p className="mt-0.5 text-[9px] tracking-wide text-lyra-mutedDim">{label}</p></div>
}

function Detail({ label, value }: { label: string; value: string | number }) {
  return <div className="rounded-lg border border-lyra-borderSoft bg-lyra-bgSoft/25 p-2.5"><p className="font-mono text-[9px] tracking-wide text-lyra-mutedDim">{label}</p><p className="mt-1 break-words text-xs text-lyra-text">{value}</p></div>
}

function ControlGroup({ label, children }: { label: string; children: ReactNode }) {
  return <div className="flex items-center gap-1 rounded-lg border border-lyra-borderSoft bg-lyra-panel/90 p-1 backdrop-blur"><span className="px-2 font-mono text-[9px] tracking-wide text-lyra-mutedDim">{label}</span>{children}</div>
}

function FilterButton({ active, onClick, label, tone }: { active: boolean; onClick: () => void; label: string; tone?: string }) {
  return <button type="button" aria-pressed={active} onClick={onClick} className={`rounded-md border px-2.5 py-1.5 text-[10px] transition ${active ? 'border-lyra-gold/35 bg-lyra-gold/10 text-lyra-goldBright' : 'border-lyra-borderSoft bg-transparent text-lyra-muted hover:border-lyra-border hover:text-lyra-text'}`}><span className="inline-flex items-center gap-1.5">{tone && <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: tone }} />}{label}</span></button>
}

function IconButton({ label, onClick, children }: { label: string; onClick: () => void; children: ReactNode }) {
  return <button type="button" aria-label={label} title={label} onClick={onClick} className="flex h-8 w-8 items-center justify-center rounded-md text-lyra-muted transition hover:bg-lyra-gold/10 hover:text-lyra-goldBright">{children}</button>
}
