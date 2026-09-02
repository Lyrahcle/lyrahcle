import type { CorrelationNode } from '../../lib/correlationGraph'
import { useI18n } from '../../lib/I18nContext'

interface NodeIndexProps {
  nodes: CorrelationNode[]
  selectedId: string | null
  routeIds: Set<string>
  onSelect: (id: string) => void
  onHover: (id: string | null) => void
  toneFor: (node: CorrelationNode) => string
}

export default function NodeIndex({ nodes, selectedId, routeIds, onSelect, onHover, toneFor }: NodeIndexProps) {
  const { t } = useI18n()

  return (
    <div className="lyra-node-index">
      <p className="lyra-node-index__title">{t('correlationGraph.nodeIndex')}</p>
      {nodes.map((node) => {
        const selected = node.id === selectedId
        return (
          <button
            key={node.id}
            type="button"
            data-correlation-node={node.id}
            aria-label={`${t(`correlationGraph.kind.${node.kind}`)}: ${node.label}`}
            aria-pressed={selected}
            tabIndex={0}
            onClick={() => onSelect(node.id)}
            onFocus={() => onHover(node.id)}
            onBlur={() => onHover(null)}
            onMouseEnter={() => onHover(node.id)}
            onMouseLeave={() => onHover(null)}
            className={`flex items-center gap-2 rounded-lg px-2 py-1.5 text-left text-[10px] transition ${
              selected
                ? 'bg-lyra-gold/14 text-lyra-goldBright'
                : routeIds.has(node.id)
                  ? 'text-lyra-text'
                  : 'text-lyra-muted hover:bg-lyra-gold/8'
            }`}
          >
            <span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ backgroundColor: toneFor(node) }} />
            <span className="min-w-0 flex-1 truncate">{node.label}</span>
          </button>
        )
      })}
    </div>
  )
}
