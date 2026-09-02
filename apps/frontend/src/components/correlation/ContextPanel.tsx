import type { ComponentType } from 'react'
import type { LucideProps } from 'lucide-react'
import { ArrowUpRight, Crosshair } from 'lucide-react'
import type { CorrelationEdge, CorrelationNode } from '../../lib/correlationGraph'
import { useI18n } from '../../lib/I18nContext'

interface RelatedItem {
  edge: CorrelationEdge
  node: CorrelationNode
}

interface ContextPanelProps {
  selected: CorrelationNode | null
  connections: number
  related: RelatedItem[]
  path: string[]
  nodeMap: Map<string, CorrelationNode>
  sourceNode: CorrelationNode | null
  targetNode: CorrelationNode | null
  tone: string
  severityTone: string
  icon: ComponentType<LucideProps>
  onOpen: () => void
  onTrace: () => void
  onClearRoute: () => void
  onSelect: (id: string) => void
  relationLabel: (relation: CorrelationEdge['relation']) => string
  description: string
  lastEvidence: string
}

function capitalize(value: string): string {
  return value ? value.charAt(0).toLocaleUpperCase() + value.slice(1) : value
}

export default function ContextPanel({
  selected,
  connections,
  related,
  path,
  nodeMap,
  sourceNode,
  targetNode,
  tone,
  severityTone,
  icon: Icon,
  onOpen,
  onTrace,
  onClearRoute,
  onSelect,
  relationLabel,
  description,
  lastEvidence,
}: ContextPanelProps) {
  const { t } = useI18n()

  if (!selected) {
    return (
      <div className="rounded-2xl border border-lyra-borderSoft/70 bg-[#080a0d]/82 p-4 backdrop-blur-md">
        <p className="text-[11px] font-medium text-lyra-text">{t('correlationGraph.noSelection')}</p>
        <p className="mt-1.5 text-[10px] leading-4 text-lyra-mutedDim">{t('correlationGraph.interactionHint')}</p>
      </div>
    )
  }

  return (
    <div className="grid gap-2.5">
      <div className="rounded-2xl border border-lyra-borderSoft/70 bg-[#080a0d]/84 p-3.5 backdrop-blur-md">
        <div className="flex items-start gap-2.5">
          <span
            className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-lyra-borderSoft/80"
            style={{ color: tone }}
          >
            <Icon className="h-[15px] w-[15px]" strokeWidth={1.6} />
          </span>
          <div className="min-w-0 flex-1">
            <p className="font-mono text-[8px] tracking-[0.16em] text-lyra-mutedDim">
              {t(`correlationGraph.kind.${selected.kind}`)}
            </p>
            <h2 className="mt-0.5 break-words text-[13px] font-medium leading-5 text-lyra-text">{selected.label}</h2>
          </div>
          {selected.severity !== 'unknown' && (
            <span
              className="shrink-0 rounded-full border px-1.5 py-0.5 font-mono text-[8px]"
              style={{ borderColor: severityTone, color: severityTone }}
            >
              {capitalize(selected.severity)}
            </span>
          )}
        </div>

        <p className="mt-2.5 line-clamp-3 text-[10px] leading-4 text-lyra-muted">{description}</p>

        <dl className="mt-3 flex items-center gap-4 border-t border-lyra-borderSoft/60 pt-2.5">
          <div>
            <dt className="font-mono text-[8px] tracking-wider text-lyra-mutedDim">
              {t('correlationGraph.connections')}
            </dt>
            <dd className="mt-0.5 font-mono text-[13px] text-lyra-text">{connections}</dd>
          </div>
          <div className="min-w-0">
            <dt className="font-mono text-[8px] tracking-wider text-lyra-mutedDim">
              {t('correlationGraph.lastEvidence')}
            </dt>
            <dd className="mt-0.5 truncate text-[10px] text-lyra-muted">{lastEvidence}</dd>
          </div>
        </dl>

        <div className="mt-3 grid grid-cols-2 gap-1.5">
          <button
            type="button"
            onClick={onOpen}
            className="inline-flex items-center justify-center gap-1 rounded-lg border border-lyra-gold/30 bg-lyra-gold/10 px-2 py-1.5 text-[10px] font-medium text-lyra-goldBright transition hover:bg-lyra-gold/16"
          >
            {t('correlationGraph.openWorkspace')} <ArrowUpRight className="h-3 w-3" aria-hidden="true" />
          </button>
          <button
            type="button"
            onClick={onTrace}
            className="inline-flex items-center justify-center gap-1 rounded-lg border border-lyra-borderSoft px-2 py-1.5 text-[10px] text-lyra-muted transition hover:border-lyra-gold/35 hover:text-lyra-text"
          >
            <Crosshair className="h-3 w-3" aria-hidden="true" /> {t('correlationGraph.traceFromHere')}
          </button>
        </div>
      </div>

      {(sourceNode || targetNode) && (
        <div className="rounded-2xl border border-lyra-gold/20 bg-lyra-gold/[0.06] p-3 backdrop-blur-md">
          <div className="flex items-center justify-between gap-2">
            <p className="font-mono text-[8px] tracking-[0.14em] text-lyra-gold">
              {t('correlationGraph.route.title')}
            </p>
            <button type="button" onClick={onClearRoute} className="text-[9px] text-lyra-muted hover:text-lyra-text">
              {t('correlationGraph.route.clear')}
            </button>
          </div>
          <p className="mt-1.5 text-[10px] text-lyra-muted">
            {sourceNode?.label || '—'} <span aria-hidden="true">→</span>{' '}
            {targetNode?.label || t('correlationGraph.route.waiting')}
          </p>
          {targetNode && !path.length && (
            <p role="status" className="mt-1.5 text-[10px] text-lyra-warn">
              {t('correlationGraph.route.noPath')}
            </p>
          )}
          {path.length > 1 && (
            <ol className="mt-2 grid gap-1">
              {path.map((id, index) => {
                const node = nodeMap.get(id)
                if (!node) return null
                return (
                  <li key={id}>
                    <button
                      type="button"
                      onClick={() => onSelect(id)}
                      className="flex w-full items-center gap-2 rounded-md px-1 py-0.5 text-left text-[10px] text-lyra-text transition hover:bg-lyra-gold/10"
                    >
                      <span className="font-mono text-[9px] text-lyra-gold">{String(index + 1).padStart(2, '0')}</span>
                      <span className="truncate">{node.label}</span>
                    </button>
                  </li>
                )
              })}
            </ol>
          )}
        </div>
      )}

      <div className="rounded-2xl border border-lyra-borderSoft/70 bg-[#080a0d]/82 p-3 backdrop-blur-md">
        <div className="mb-1.5 flex items-center justify-between gap-2">
          <p className="font-mono text-[8px] tracking-[0.14em] text-lyra-mutedDim">
            {t('correlationGraph.whyConnected')}
          </p>
          <span className="font-mono text-[9px] text-lyra-mutedDim">{related.length}</span>
        </div>
        <div className="grid max-h-56 gap-1 overflow-y-auto">
          {related.length ? (
            related.slice(0, 10).map(({ edge, node }) => (
              <button
                key={edge.id}
                type="button"
                onClick={() => onSelect(node.id)}
                className="group rounded-lg px-2 py-1.5 text-left transition hover:bg-lyra-gold/8"
              >
                <span className="flex items-center justify-between gap-2">
                  <span className="truncate text-[10px] text-lyra-text group-hover:text-lyra-goldBright">
                    {node.label}
                  </span>
                  <span className="font-mono text-[8px] text-lyra-mutedDim">{edge.count > 1 ? `×${edge.count}` : ''}</span>
                </span>
                <span className="mt-0.5 block truncate text-[9px] text-lyra-mutedDim">{relationLabel(edge.relation)}</span>
              </button>
            ))
          ) : (
            <p className="px-2 py-1 text-[10px] text-lyra-mutedDim">{t('correlationGraph.noRelations')}</p>
          )}
        </div>
      </div>
    </div>
  )
}
