import { useEffect, useMemo, useRef, useState } from 'react'
import {
  Activity,
  Boxes,
  Crosshair,
  Filter,
  Focus,
  Layers,
  Orbit,
  Radar,
  RadioTower,
  RotateCcw,
  ShieldAlert,
  Sparkles,
} from 'lucide-react'
import { useI18n } from '../lib/I18nContext'
import { lazyComponent } from '../lib/lazyComponent'
import type { RouteTab } from '../lib/routeTabs'
import {
  buildCorrelationGraph,
  chooseCorrelationAnchor,
  correlationNeighborhood,
  graphAdjacency,
  graphDegrees,
  shortestCorrelationPath,
  type CorrelationEdge,
  type CorrelationGraph as CorrelationGraphModel,
  type CorrelationNode,
  type CorrelationNodeKind,
  type CorrelationRecord,
  type CorrelationSeverity,
} from '../lib/correlationGraph'
import { buildOrbitalLayout, pathEdgeKeys, LAYOUT_KIND_ORDER } from '../lib/correlationLayout3d'
import ContextPanel from './correlation/ContextPanel'
import FilterPopover, { type SeverityFilter, type WindowKey } from './correlation/FilterPopover'
import { HudButton, HudDivider, HudRail } from './correlation/HudPrimitives'
import NodeIndex from './correlation/NodeIndex'
import NodeSearch from './correlation/NodeSearch'
import { prefersReducedMotion, supportsWebGl } from './correlation/capabilities'
import { kindColor, readScenePalette, severityColor } from './correlation/sceneTheme'
import StatusState from './StatusState'

const OrbitalScene = lazyComponent(
  () => import('./correlation/OrbitalScene'),
  <div className="absolute inset-0 animate-pulse rounded-2xl bg-[#080a0d]/40" aria-hidden="true" />,
)

const DAY_MS = 24 * 60 * 60 * 1000
const KIND_ORDER = LAYOUT_KIND_ORDER
const SEVERITY_RANK: Record<CorrelationSeverity, number> = { unknown: 0, low: 1, medium: 2, high: 3, critical: 4 }
const MAX_SCENE_NODES = 64

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

function nodeDescription(node: CorrelationNode): string {
  const values = [node.subtitle, node.metadata?.status, node.metadata?.kind, node.metadata?.metric]
  return values.find((value) => String(value ?? '').trim()) || '—'
}

function formatTime(value?: string): string {
  if (!value) return '—'
  const parsed = Date.parse(value)
  return Number.isFinite(parsed) ? new Date(parsed).toLocaleString() : '—'
}

export default function CorrelationGraph({
  satellites,
  agents,
  readings,
  anomalies,
  incidents,
  assets,
  programs,
  alerts,
  cves,
  access = {},
  onNavigate,
  onOpenSatellite,
  now,
}: CorrelationGraphProps) {
  const { t } = useI18n()
  const [mountedAt] = useState(Date.now)
  const graphNow = now ?? mountedAt
  const [timeWindow, setTimeWindow] = useState<WindowKey>('7d')
  const [depth, setDepth] = useState(2)
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all')
  const [enabledKinds, setEnabledKinds] = useState<Set<CorrelationNodeKind>>(() => new Set(KIND_ORDER))
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [routeOriginId, setRouteOriginId] = useState<string | null>(null)
  const [routeTargetId, setRouteTargetId] = useState<string | null>(null)
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [focusMode, setFocusMode] = useState(false)
  const [showClusters, setShowClusters] = useState(true)
  const [autoRotate, setAutoRotate] = useState(true)
  const [webgl] = useState(supportsWebGl)
  const [reducedMotion] = useState(prefersReducedMotion)
  const palette = useMemo(() => readScenePalette(), [])
  const interactionRef = useRef(false)

  const graph = useMemo(
    () =>
      buildCorrelationGraph({
        satellites,
        agents,
        readings,
        anomalies,
        incidents,
        assets,
        programs,
        alerts,
        cves,
        now: graphNow,
        windowMs: windowMs(timeWindow),
      }),
    [satellites, agents, readings, anomalies, incidents, assets, programs, alerts, cves, graphNow, timeWindow],
  )

  const nodeMap = useMemo(() => new Map(graph.nodes.map((node) => [node.id, node])), [graph.nodes])
  const degrees = useMemo(() => graphDegrees(graph), [graph])
  const adjacency = useMemo(() => graphAdjacency(graph), [graph])

  useEffect(() => {
    if (selectedId && nodeMap.has(selectedId)) return
    setSelectedId(chooseCorrelationAnchor(graph))
    setRouteOriginId(null)
    setRouteTargetId(null)
  }, [graph, nodeMap, selectedId])

  const selected = selectedId ? nodeMap.get(selectedId) ?? null : null
  const path = useMemo(
    () => (routeOriginId && routeTargetId ? shortestCorrelationPath(graph, routeOriginId, routeTargetId) : []),
    [graph, routeOriginId, routeTargetId],
  )
  const routeIds = useMemo(() => new Set(path), [path])
  const routeKeys = useMemo(() => pathEdgeKeys(path), [path])

  const filteredGraph = useMemo<CorrelationGraphModel>(() => {
    const nodes = graph.nodes.filter((node) => enabledKinds.has(node.kind) && severityMatches(node, severityFilter))
    if (selected && !nodes.some((node) => node.id === selected.id)) nodes.push(selected)
    const ids = new Set(nodes.map((node) => node.id))
    return { ...graph, nodes, edges: graph.edges.filter((edge) => ids.has(edge.source) && ids.has(edge.target)) }
  }, [graph, enabledKinds, severityFilter, selected])

  const visibleIds = useMemo(() => {
    if (!selectedId) return new Set<string>()
    const result = correlationNeighborhood(filteredGraph, selectedId, depth, MAX_SCENE_NODES)
    const contextTarget = Math.min(30, filteredGraph.nodes.length)
    if (result.size < Math.min(12, contextTarget)) {
      const ranked = [...filteredGraph.nodes].sort(
        (a, b) => (degrees.get(b.id) ?? 0) - (degrees.get(a.id) ?? 0) || a.label.localeCompare(b.label),
      )
      for (const node of ranked) {
        result.add(node.id)
        if (result.size >= contextTarget) break
      }
    }
    for (const id of path) result.add(id)
    return result
  }, [filteredGraph, selectedId, depth, path, degrees])

  const layout = useMemo(
    () => buildOrbitalLayout(graph, selectedId, visibleIds),
    [graph, selectedId, visibleIds],
  )

  const visibleEdges = useMemo(
    () => graph.edges.filter((edge) => layout.byId.has(edge.source) && layout.byId.has(edge.target)),
    [graph.edges, layout],
  )

  const neighborIds = useMemo(() => {
    if (!selectedId) return new Set<string>()
    const result = new Set<string>([selectedId])
    for (const id of adjacency.get(selectedId) ?? []) result.add(id)
    return result
  }, [adjacency, selectedId])

  const orderedNodes = useMemo(
    () =>
      [...layout.placements]
        .sort((a, b) => a.ring - b.ring || b.degree - a.degree || a.node.label.localeCompare(b.node.label))
        .map((placement) => placement.node),
    [layout.placements],
  )

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
  const focusToken = `${selectedId ?? 'none'}:${depth}:${layout.placements.length}`

  function selectNode(id: string) {
    setSelectedId(id)
    if (routeOriginId && id !== routeOriginId) setRouteTargetId(id)
    const node = nodeMap.get(id)
    if (node && !enabledKinds.has(node.kind)) setEnabledKinds((current) => new Set([...current, node.kind]))
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
    setFocusMode(false)
    setShowClusters(true)
    setAutoRotate(true)
    setFiltersOpen(false)
    interactionRef.current = false
    clearRoute()
    setSelectedId(chooseCorrelationAnchor(graph))
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

  const SelectedIcon = nodeIcon(selected?.kind ?? 'satellite')

  return (
    <section aria-labelledby="correlation-graph-heading" className="relative">
      <div className="relative min-h-[720px] overflow-hidden rounded-2xl border border-lyra-borderSoft bg-[#05070a] xl:min-h-[calc(100vh-118px)]">
        <div className="pointer-events-none absolute inset-0 lyra-correlation-stage" />

        <div
          role="group"
          aria-label={t('correlationGraph.canvasLabel')}
          className="absolute inset-0 xl:left-[186px] xl:right-[324px]"
        >
          {webgl ? (
            <OrbitalScene
              layout={layout}
              edges={visibleEdges}
              selectedId={selectedId}
              hoveredId={hoveredId}
              path={path}
              routeKeys={routeKeys}
              neighborIds={neighborIds}
              isolate={focusMode}
              showClusters={showClusters}
              autoRotate={autoRotate && !interactionRef.current}
              reducedMotion={reducedMotion}
              focus={layout.byId.get(selectedId ?? '')?.position ?? null}
              focusToken={focusToken}
              onSelect={selectNode}
              onHover={setHoveredId}
              onBackground={() => setHoveredId(null)}
              onUserInteract={() => {
                interactionRef.current = true
                setAutoRotate(false)
              }}
              hopLabel={(index) => t('correlationGraph.hop', { count: index })}
            />
          ) : (
            <div className="flex h-full items-center justify-center p-8">
              <p className="max-w-sm text-center text-[11px] leading-5 text-lyra-mutedDim">
                {t('correlationGraph.webglFallback')}
              </p>
            </div>
          )}
        </div>

        <div className="pointer-events-none absolute left-5 top-4 z-20 max-w-md">
          <div className="font-mono text-[8px] tracking-[0.24em] text-lyra-mutedDim">
            {t('correlationGraph.eyebrow')}
          </div>
          <h1
            id="correlation-graph-heading"
            className="mt-1 font-display text-lg font-medium tracking-tight text-lyra-text sm:text-xl"
          >
            {t('correlationGraph.title')}
          </h1>
        </div>

        <div className="absolute right-4 top-4 z-30 flex flex-col items-end gap-2">
          <HudRail>
            <HudButton
              label={t('correlationGraph.focusMode')}
              active={focusMode}
              onClick={() => setFocusMode((value) => !value)}
            >
              <Focus className="h-3.5 w-3.5" />
            </HudButton>
            <HudButton
              label={t('correlationGraph.clusters')}
              active={showClusters}
              onClick={() => setShowClusters((value) => !value)}
            >
              <Layers className="h-3.5 w-3.5" />
            </HudButton>
            <HudButton
              label={t('correlationGraph.autoRotate')}
              active={autoRotate && !interactionRef.current}
              onClick={() => {
                interactionRef.current = false
                setAutoRotate((value) => !value)
              }}
            >
              <Sparkles className="h-3.5 w-3.5" />
            </HudButton>
            <HudButton
              label={t('correlationGraph.recenter')}
              onClick={() => {
                interactionRef.current = false
                if (selectedId) setSelectedId(selectedId)
                setDepth((value) => value)
              }}
            >
              <Crosshair className="h-3.5 w-3.5" />
            </HudButton>
            <HudDivider />
            <HudButton
              label={t('correlationGraph.filters')}
              active={filtersOpen}
              onClick={() => setFiltersOpen((value) => !value)}
            >
              <Filter className="h-3.5 w-3.5" />
            </HudButton>
            <HudButton label={t('correlationGraph.reset')} onClick={resetView}>
              <RotateCcw className="h-3.5 w-3.5" />
            </HudButton>
          </HudRail>
          <FilterPopover
            open={filtersOpen}
            onClose={() => setFiltersOpen(false)}
            kinds={KIND_ORDER}
            enabledKinds={enabledKinds}
            onToggleKind={toggleKind}
            timeWindow={timeWindow}
            onTimeWindow={setTimeWindow}
            severity={severityFilter}
            onSeverity={setSeverityFilter}
            depth={depth}
            onDepth={setDepth}
            toneFor={(kind) => kindColor(kind, palette)}
          />
        </div>

        <aside
          aria-label={t('correlationGraph.contextPanel')}
          className="absolute inset-x-4 bottom-[68px] z-20 max-h-[44%] overflow-y-auto xl:inset-x-auto xl:bottom-4 xl:right-4 xl:top-[60px] xl:max-h-none xl:w-[300px]"
        >
          <ContextPanel
            selected={selected}
            connections={selected ? degrees.get(selected.id) ?? 0 : 0}
            related={related}
            path={path}
            nodeMap={nodeMap}
            sourceNode={sourceNode}
            targetNode={targetNode}
            tone={kindColor(selected?.kind ?? 'satellite', palette)}
            severityTone={severityColor(selected?.severity ?? 'unknown', palette)}
            icon={SelectedIcon}
            onOpen={openSelected}
            onTrace={startRoute}
            onClearRoute={clearRoute}
            onSelect={selectNode}
            relationLabel={(relation) => t(`correlationGraph.relations.${relation}`)}
            description={selected ? nodeDescription(selected) : '—'}
            lastEvidence={formatTime(selected?.timestamp)}
          />
        </aside>

        <NodeIndex
          nodes={orderedNodes}
          selectedId={selectedId}
          routeIds={routeIds}
          onSelect={selectNode}
          onHover={setHoveredId}
          toneFor={(node) => kindColor(node.kind, palette)}
        />

        <div className="absolute bottom-4 left-4 z-30 flex max-w-[calc(100%-32px)] flex-wrap items-center gap-2 xl:left-[186px] xl:right-[324px]">
          <NodeSearch
            nodes={graph.nodes}
            degrees={degrees}
            toneFor={(node) => kindColor(node.kind, palette)}
            kindLabel={(node) => t(`correlationGraph.kind.${node.kind}`)}
            onSelect={selectNode}
          />
          <div className="hidden items-center gap-3 rounded-full border border-lyra-borderSoft/70 bg-[#080a0d]/78 px-3.5 py-2 font-mono text-[9px] text-lyra-mutedDim backdrop-blur-md sm:flex">
            <span>
              <span className="text-lyra-text">{graph.nodes.length}</span> {t('correlationGraph.metrics.nodes')}
            </span>
            <span>
              <span className="text-lyra-text">{graph.edges.length}</span> {t('correlationGraph.metrics.links')}
            </span>
            <span>
              <span className="text-lyra-text">{graph.evidenceCount}</span> {t('correlationGraph.metrics.evidence')}
            </span>
            <span className="text-lyra-mutedDim/70">
              {t('correlationGraph.visible', { visible: layout.placements.length, total: graph.nodes.length })}
            </span>
          </div>
        </div>

        {graph.edges.length === 0 && (
          <div className="pointer-events-none absolute left-1/2 top-6 z-20 -translate-x-1/2 rounded-full border border-lyra-borderSoft/70 bg-[#080a0d]/80 px-3 py-1 text-[10px] text-lyra-mutedDim backdrop-blur">
            {t('correlationGraph.awaitingEvidence', { count: graph.nodes.length })}
          </div>
        )}

        <p className="sr-only" role="status" aria-live="polite">
          {selected
            ? t('correlationGraph.selectionAnnouncement', {
                kind: t(`correlationGraph.kind.${selected.kind}`),
                label: selected.label,
                connections: degrees.get(selected.id) ?? 0,
              })
            : t('correlationGraph.noSelection')}
        </p>

        {routeOriginId && !routeTargetId && (
          <div
            role="status"
            className="pointer-events-none absolute bottom-4 left-1/2 z-20 hidden -translate-x-1/2 rounded-full border border-lyra-gold/25 bg-[#080a0d]/86 px-3 py-1.5 text-[10px] text-lyra-goldBright backdrop-blur lg:block"
          >
            {t('correlationGraph.selectDestination', { source: sourceNode?.label || '—' })}
          </div>
        )}
      </div>

    </section>
  )
}
