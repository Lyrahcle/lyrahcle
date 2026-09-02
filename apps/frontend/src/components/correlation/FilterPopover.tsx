import { useEffect, useRef } from 'react'
import type { CorrelationNodeKind } from '../../lib/correlationGraph'
import { useI18n } from '../../lib/I18nContext'
import { HudChip } from './HudPrimitives'

export type WindowKey = '24h' | '7d' | '30d' | 'all'
export type SeverityFilter = 'all' | 'high' | 'critical'

interface FilterPopoverProps {
  open: boolean
  onClose: () => void
  kinds: CorrelationNodeKind[]
  enabledKinds: Set<CorrelationNodeKind>
  onToggleKind: (kind: CorrelationNodeKind) => void
  timeWindow: WindowKey
  onTimeWindow: (value: WindowKey) => void
  severity: SeverityFilter
  onSeverity: (value: SeverityFilter) => void
  depth: number
  onDepth: (value: number) => void
  toneFor: (kind: CorrelationNodeKind) => string
}

export default function FilterPopover({
  open,
  onClose,
  kinds,
  enabledKinds,
  onToggleKind,
  timeWindow,
  onTimeWindow,
  severity,
  onSeverity,
  depth,
  onDepth,
  toneFor,
}: FilterPopoverProps) {
  const { t } = useI18n()
  const container = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handleKey(event: globalThis.KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      ref={container}
      role="group"
      aria-label={t('correlationGraph.filters')}
      className="grid w-[268px] gap-3 rounded-2xl border border-lyra-borderSoft/80 bg-[#080a0d]/94 p-3 shadow-2xl shadow-black/50 backdrop-blur-md"
    >
      <div>
        <p className="mb-1.5 font-mono text-[8px] tracking-[0.14em] text-lyra-mutedDim">
          {t('correlationGraph.timeWindow')}
        </p>
        <div className="flex flex-wrap gap-1">
          {(['24h', '7d', '30d', 'all'] as WindowKey[]).map((value) => (
            <HudChip
              key={value}
              label={t(`correlationGraph.window.${value}`)}
              active={timeWindow === value}
              onClick={() => onTimeWindow(value)}
            />
          ))}
        </div>
      </div>

      <div>
        <p className="mb-1.5 font-mono text-[8px] tracking-[0.14em] text-lyra-mutedDim">
          {t('correlationGraph.severity')}
        </p>
        <div className="flex flex-wrap gap-1">
          {(['all', 'high', 'critical'] as SeverityFilter[]).map((value) => (
            <HudChip
              key={value}
              label={t(`correlationGraph.severityFilter.${value}`)}
              active={severity === value}
              onClick={() => onSeverity(value)}
            />
          ))}
        </div>
      </div>

      <div>
        <p className="mb-1.5 font-mono text-[8px] tracking-[0.14em] text-lyra-mutedDim">
          {t('correlationGraph.depth')}
        </p>
        <div className="flex flex-wrap gap-1">
          {[1, 2, 3].map((value) => (
            <HudChip key={value} label={`${value}`} active={depth === value} onClick={() => onDepth(value)} />
          ))}
        </div>
      </div>

      <div>
        <p className="mb-1.5 font-mono text-[8px] tracking-[0.14em] text-lyra-mutedDim">
          {t('correlationGraph.nodeTypes')}
        </p>
        <div className="flex flex-wrap gap-1">
          {kinds.map((kind) => (
            <HudChip
              key={kind}
              label={t(`correlationGraph.kind.${kind}`)}
              active={enabledKinds.has(kind)}
              onClick={() => onToggleKind(kind)}
              tone={toneFor(kind)}
            />
          ))}
        </div>
      </div>
    </div>
  )
}
