import { useMemo } from 'react'
import { useI18n } from '../lib/I18nContext'
import { classifyRFAttackCandidates } from './RFAttackTriage'

type RFAnomaly = {
  id?: string
  agent_id?: string
  metric?: string
  z_score?: number | string
  detected_at?: string
}

type Incident = {
  id?: string
  title?: string
  description?: string
  severity?: string
  status?: string
}

type ThreatCell = {
  id: string
  tactic: string
  technique: string
  evidenceCount: number
  evidence: string[]
  state: 'observed' | 'candidate' | 'unavailable'
}

function incidentText(incident: Incident) {
  return `${incident.title || ''} ${incident.description || ''}`.toLowerCase()
}

export function buildSpaceThreatMatrix(anomalies: RFAnomaly[] = [], incidents: Incident[] = []): ThreatCell[] {
  const candidates = classifyRFAttackCandidates(anomalies)
  const jamming = candidates.filter((item) => item.kind === 'jamming')
  const spoofing = candidates.filter((item) => item.kind === 'spoofing')

  const access = incidents.filter((incident) =>
    /(unauthori[sz]ed|credential|authentication|login|account|access|compromis)/.test(incidentText(incident)),
  )
  const exploitation = incidents.filter((incident) =>
    /(cve-|vulnerab|exploit|remote code|code execution|injection)/.test(incidentText(incident)),
  )

  return [
    {
      id: 'rf-interference',
      tactic: 'effects',
      technique: 'rfInterference',
      evidenceCount: jamming.length,
      evidence: jamming.map((item) => `${item.agentId} · ${item.evidence}`),
      state: jamming.length ? 'candidate' : 'unavailable',
    },
    {
      id: 'signal-spoofing',
      tactic: 'initialAccess',
      technique: 'signalSpoofing',
      evidenceCount: spoofing.length,
      evidence: spoofing.map((item) => `${item.agentId} · ${item.evidence}`),
      state: spoofing.length ? 'candidate' : 'unavailable',
    },
    {
      id: 'replay',
      tactic: 'commandControl',
      technique: 'replay',
      evidenceCount: 0,
      evidence: [],
      state: 'unavailable',
    },
    {
      id: 'ground-access',
      tactic: 'initialAccess',
      technique: 'groundAccess',
      evidenceCount: access.length,
      evidence: access.slice(0, 5).map((incident) => incident.title || incident.id || 'incident'),
      state: access.length ? 'observed' : 'unavailable',
    },
    {
      id: 'software-exploitation',
      tactic: 'execution',
      technique: 'softwareExploitation',
      evidenceCount: exploitation.length,
      evidence: exploitation.slice(0, 5).map((incident) => incident.title || incident.id || 'incident'),
      state: exploitation.length ? 'observed' : 'unavailable',
    },
  ]
}

function stateClass(state: ThreatCell['state']) {
  if (state === 'observed') return 'border-lyra-bad/40 bg-lyra-bad/10'
  if (state === 'candidate') return 'border-lyra-warn/40 bg-lyra-warn/10'
  return 'border-lyra-borderSoft bg-lyra-bgSoft/20'
}

export default function SpaceThreatMatrix({
  anomalies = [],
  incidents = [],
}: {
  anomalies?: RFAnomaly[]
  incidents?: Incident[]
}) {
  const { t } = useI18n()
  const cells = useMemo(() => buildSpaceThreatMatrix(anomalies, incidents), [anomalies, incidents])
  const evidenced = cells.filter((cell) => cell.evidenceCount > 0).length

  return (
    <section aria-labelledby="space-threat-matrix-title" className="grid gap-4">
      <div>
        <h3 id="space-threat-matrix-title" className="text-sm font-medium text-lyra-text">
          {t('spaceThreatMatrix.title')}
        </h3>
        <p className="mt-1 text-xs text-lyra-muted">{t('spaceThreatMatrix.subtitle')}</p>
      </div>

      <div className="flex flex-wrap gap-2 text-[11px] text-lyra-mutedDim">
        <span className="rounded border border-lyra-borderSoft px-2 py-1">
          {t('spaceThreatMatrix.evidenced', { count: evidenced })}
        </span>
        <span className="rounded border border-lyra-borderSoft px-2 py-1">
          {t('spaceThreatMatrix.total', { count: cells.length })}
        </span>
      </div>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {cells.map((cell) => (
          <article key={cell.id} className={`rounded border p-4 ${stateClass(cell.state)}`}>
            <p className="text-[10px] tracking-[0.12em] text-lyra-mutedDim">
              {t(`spaceThreatMatrix.tactics.${cell.tactic}`)}
            </p>
            <h4 className="mt-1 text-sm font-medium text-lyra-text">
              {t(`spaceThreatMatrix.techniques.${cell.technique}`)}
            </h4>
            <p className="mt-2 text-xs text-lyra-muted">
              {cell.state === 'candidate'
                ? t('spaceThreatMatrix.candidate', { count: cell.evidenceCount })
                : cell.state === 'observed'
                  ? t('spaceThreatMatrix.observed', { count: cell.evidenceCount })
                  : t('spaceThreatMatrix.unavailable')}
            </p>
            {cell.evidence.length ? (
              <ul className="mt-3 grid gap-1">
                {cell.evidence.slice(0, 3).map((item, index) => (
                  <li key={`${cell.id}-${index}`} className="truncate font-mono text-[10px] text-lyra-mutedDim">
                    {item}
                  </li>
                ))}
              </ul>
            ) : null}
          </article>
        ))}
      </div>

      <p className="text-[10px] text-lyra-mutedDim">{t('spaceThreatMatrix.disclaimer')}</p>
    </section>
  )
}
