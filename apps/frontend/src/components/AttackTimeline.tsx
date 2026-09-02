import { useMemo, useState } from 'react'
import { useI18n } from '../lib/I18nContext'

type Entity = Record<string, any>
type TimelineKind = 'capture' | 'anomaly' | 'incident'

type TimelineEvent = {
  id: string
  kind: TimelineKind
  timestamp: string
  title: string
  detail: string
  agentId?: string
  satelliteId?: string
  severity?: string
}

function timestampOf(value: unknown): number {
  const parsed = Date.parse(String(value || ''))
  return Number.isFinite(parsed) ? parsed : 0
}

function firstText(item: Entity, keys: string[]): string | undefined {
  for (const key of keys) {
    const value = item?.[key]
    if (value !== undefined && value !== null && String(value).trim()) return String(value)
  }
  return undefined
}

function numberText(value: unknown, digits = 1): string | null {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed.toFixed(digits) : null
}

export function buildAttackTimeline(
  readings: Entity[] = [],
  anomalies: Entity[] = [],
  incidents: Entity[] = [],
): TimelineEvent[] {
  const captures: TimelineEvent[] = readings.flatMap((reading, index) => {
    const timestamp = firstText(reading, ['captured_at', 'observed_at', 'created_at', 'timestamp'])
    if (!timestamp || !timestampOf(timestamp)) return []
    const frequencyHz = Number(reading.frequency_hz)
    const power = numberText(reading.power_dbm)
    const frequency = Number.isFinite(frequencyHz) ? `${(frequencyHz / 1_000_000).toFixed(6)} MHz` : null
    return [{
      id: `capture-${reading.id || index}`,
      kind: 'capture' as const,
      timestamp,
      title: 'RF capture',
      detail: [frequency, power === null ? null : `${power} dBm`].filter(Boolean).join(' · ') || 'Captured RF evidence',
      agentId: firstText(reading, ['agent_id']),
      satelliteId: firstText(reading, ['satellite_id']),
    }]
  })

  const anomalyEvents: TimelineEvent[] = anomalies.flatMap((anomaly, index) => {
    const timestamp = firstText(anomaly, ['detected_at', 'created_at', 'timestamp'])
    if (!timestamp || !timestampOf(timestamp)) return []
    const zScore = numberText(anomaly.z_score)
    const metric = firstText(anomaly, ['metric']) || 'RF anomaly'
    return [{
      id: `anomaly-${anomaly.id || index}`,
      kind: 'anomaly' as const,
      timestamp,
      title: 'RF anomaly',
      detail: zScore === null ? metric : `${metric} · ${zScore}σ`,
      agentId: firstText(anomaly, ['agent_id']),
      satelliteId: firstText(anomaly, ['satellite_id']),
      severity: firstText(anomaly, ['severity']),
    }]
  })

  const incidentEvents: TimelineEvent[] = incidents.flatMap((incident, index) => {
    const timestamp = firstText(incident, ['created_at', 'opened_at', 'updated_at', 'timestamp'])
    if (!timestamp || !timestampOf(timestamp)) return []
    return [{
      id: `incident-${incident.id || index}`,
      kind: 'incident' as const,
      timestamp,
      title: firstText(incident, ['title', 'name']) || 'Security incident',
      detail: firstText(incident, ['summary', 'description', 'status']) || 'Incident evidence',
      agentId: firstText(incident, ['agent_id']),
      satelliteId: firstText(incident, ['satellite_id']),
      severity: firstText(incident, ['severity']),
    }]
  })

  return [...captures, ...anomalyEvents, ...incidentEvents]
    .sort((a, b) => timestampOf(b.timestamp) - timestampOf(a.timestamp))
    .slice(0, 60)
}

function kindClass(kind: TimelineKind) {
  if (kind === 'incident') return 'border-lyra-bad/40 bg-lyra-bad/10 text-lyra-bad'
  if (kind === 'anomaly') return 'border-lyra-warn/40 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-border bg-lyra-panel text-lyra-muted'
}

export default function AttackTimeline({ readings = [], anomalies = [], incidents = [] }: {
  readings?: Entity[]
  anomalies?: Entity[]
  incidents?: Entity[]
}) {
  const { t } = useI18n()
  const [kind, setKind] = useState<'all' | TimelineKind>('all')
  const events = useMemo(() => buildAttackTimeline(readings, anomalies, incidents), [readings, anomalies, incidents])
  const visible = kind === 'all' ? events : events.filter((event) => event.kind === kind)

  const counts = useMemo(() => ({
    capture: events.filter((event) => event.kind === 'capture').length,
    anomaly: events.filter((event) => event.kind === 'anomaly').length,
    incident: events.filter((event) => event.kind === 'incident').length,
  }), [events])

  return (
    <section aria-labelledby="attack-timeline-title" className="grid gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h3 id="attack-timeline-title" className="text-sm font-medium text-lyra-text">{t('attackTimeline.title')}</h3>
          <p className="mt-1 text-xs text-lyra-muted">{t('attackTimeline.subtitle')}</p>
        </div>
        <label className="text-xs text-lyra-muted">
          <span className="mb-1 block">{t('attackTimeline.filter')}</span>
          <select aria-label={t('attackTimeline.filter')} value={kind} onChange={(event) => setKind(event.target.value as typeof kind)} className="rounded border border-lyra-border bg-lyra-panel px-2 py-1.5 text-xs text-lyra-text">
            <option value="all">{t('attackTimeline.all')}</option>
            <option value="capture">{t('attackTimeline.capture')}</option>
            <option value="anomaly">{t('attackTimeline.anomaly')}</option>
            <option value="incident">{t('attackTimeline.incident')}</option>
          </select>
        </label>
      </div>

      <div className="grid gap-2 sm:grid-cols-3">
        <p className="rounded border border-lyra-borderSoft px-3 py-2 text-xs text-lyra-muted"><span className="font-mono text-lyra-text">{counts.capture}</span> {t('attackTimeline.captures')}</p>
        <p className="rounded border border-lyra-borderSoft px-3 py-2 text-xs text-lyra-muted"><span className="font-mono text-lyra-text">{counts.anomaly}</span> {t('attackTimeline.anomalies')}</p>
        <p className="rounded border border-lyra-borderSoft px-3 py-2 text-xs text-lyra-muted"><span className="font-mono text-lyra-text">{counts.incident}</span> {t('attackTimeline.incidents')}</p>
      </div>

      {!visible.length ? (
        <div className="rounded border border-dashed border-lyra-border px-5 py-8 text-center text-sm text-lyra-mutedDim">{t('attackTimeline.empty')}</div>
      ) : (
        <ol className="grid gap-2" aria-label={t('attackTimeline.title')}>
          {visible.map((event) => (
            <li key={event.id} className="grid grid-cols-[auto_1fr] gap-3 rounded border border-lyra-borderSoft bg-lyra-bg/30 p-3">
              <span className={`mt-0.5 h-fit rounded border px-2 py-1 text-[10px] ${kindClass(event.kind)}`}>{t(`attackTimeline.kind.${event.kind}`)}</span>
              <div className="min-w-0">
                <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                  <p className="text-sm text-lyra-text">{event.kind === 'capture' ? t('attackTimeline.capture') : event.kind === 'anomaly' ? t('attackTimeline.anomaly') : event.title}</p>
                  <time dateTime={event.timestamp} className="font-mono text-[10px] text-lyra-mutedDim">{new Date(event.timestamp).toLocaleString()}</time>
                </div>
                <p className="mt-1 font-mono text-[11px] text-lyra-mutedDim">{event.detail}</p>
                {(event.agentId || event.satelliteId || event.severity) && (
                  <p className="mt-1 text-[10px] text-lyra-mutedDim">{[event.agentId, event.satelliteId, event.severity].filter(Boolean).join(' · ')}</p>
                )}
              </div>
            </li>
          ))}
        </ol>
      )}

      <p className="text-[10px] text-lyra-mutedDim">{t('attackTimeline.disclaimer')}</p>
    </section>
  )
}
