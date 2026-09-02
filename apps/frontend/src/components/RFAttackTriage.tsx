import { useMemo } from 'react'
import { useI18n } from '../lib/I18nContext'

type RFAnomaly = {
  id?: string
  agent_id?: string
  metric?: string
  value?: number | string
  baseline_mean?: number | string
  z_score?: number | string
  detected_at?: string
}

type AttackKind = 'jamming' | 'spoofing'

type AttackCandidate = {
  id: string
  kind: AttackKind
  confidence: 'high' | 'medium'
  agentId: string
  zScore: number
  detectedAt?: string
  evidence: string
}

function finiteNumber(value: unknown): number | null {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

export function classifyRFAttackCandidates(anomalies: RFAnomaly[] = []): AttackCandidate[] {
  const candidates: AttackCandidate[] = []

  anomalies.forEach((anomaly, index) => {
    const metric = String(anomaly?.metric || '')
    const zScore = finiteNumber(anomaly?.z_score)
    if (zScore === null || Math.abs(zScore) < 4) return

    const confidence: AttackCandidate['confidence'] = Math.abs(zScore) >= 5 ? 'high' : 'medium'
    const agentId = anomaly?.agent_id || '—'
    const base = anomaly?.id || `${metric}-${index}`

    if (metric === 'power_dbm' || metric === 'power_dbm_trend') {
      if (zScore > 0) {
        candidates.push({
          id: `jamming-${base}`,
          kind: 'jamming',
          confidence,
          agentId,
          zScore,
          detectedAt: anomaly?.detected_at,
          evidence: `${metric} · +${zScore.toFixed(1)}σ`,
        })
      }
      return
    }

    if (metric === 'frequency_hz' || metric === 'frequency_hz_trend') {
      candidates.push({
        id: `spoofing-${base}`,
        kind: 'spoofing',
        confidence,
        agentId,
        zScore,
        detectedAt: anomaly?.detected_at,
        evidence: `${metric} · ${zScore.toFixed(1)}σ`,
      })
    }
  })

  return candidates.sort((a, b) => Math.abs(b.zScore) - Math.abs(a.zScore))
}

function badgeTone(confidence: AttackCandidate['confidence']) {
  return confidence === 'high'
    ? 'border-lyra-bad/40 bg-lyra-bad/10 text-lyra-bad'
    : 'border-lyra-warn/40 bg-lyra-warn/10 text-lyra-warn'
}

export default function RFAttackTriage({ anomalies = [] }: { anomalies?: RFAnomaly[] }) {
  const { t } = useI18n()
  const candidates = useMemo(() => classifyRFAttackCandidates(anomalies), [anomalies])
  const jamming = candidates.filter((candidate) => candidate.kind === 'jamming')
  const spoofing = candidates.filter((candidate) => candidate.kind === 'spoofing')

  return (
    <section aria-labelledby="rf-attack-triage-title" className="grid gap-4">
      <div>
        <h3 id="rf-attack-triage-title" className="text-sm font-medium text-lyra-text">{t('rfAttackTriage.title')}</h3>
        <p className="mt-1 text-xs text-lyra-muted">{t('rfAttackTriage.subtitle')}</p>
      </div>

      <div className="grid gap-3 md:grid-cols-3">
        <div className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4">
          <p className="text-xs text-lyra-mutedDim">{t('rfAttackTriage.jamming')}</p>
          <p className="mt-1 font-mono text-2xl text-lyra-text">{jamming.length}</p>
          <p className="mt-1 text-[11px] text-lyra-mutedDim">{t('rfAttackTriage.jammingHint')}</p>
        </div>
        <div className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4">
          <p className="text-xs text-lyra-mutedDim">{t('rfAttackTriage.spoofing')}</p>
          <p className="mt-1 font-mono text-2xl text-lyra-text">{spoofing.length}</p>
          <p className="mt-1 text-[11px] text-lyra-mutedDim">{t('rfAttackTriage.spoofingHint')}</p>
        </div>
        <div className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4">
          <p className="text-xs text-lyra-mutedDim">{t('rfAttackTriage.replay')}</p>
          <p className="mt-1 font-mono text-2xl text-lyra-mutedDim">—</p>
          <p className="mt-1 text-[11px] text-lyra-mutedDim">{t('rfAttackTriage.replayUnavailable')}</p>
        </div>
      </div>

      {!candidates.length ? (
        <div className="rounded border border-dashed border-lyra-border px-5 py-8 text-center text-sm text-lyra-mutedDim">
          {t('rfAttackTriage.empty')}
        </div>
      ) : (
        <div className="grid gap-2">
          {candidates.slice(0, 8).map((candidate) => (
            <article key={candidate.id} className="flex flex-col gap-2 rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="min-w-0">
                <p className="text-sm text-lyra-text">{candidate.kind === 'jamming' ? t('rfAttackTriage.jammingCandidate') : t('rfAttackTriage.spoofingCandidate')}</p>
                <p className="mt-1 truncate font-mono text-[11px] text-lyra-mutedDim">{candidate.agentId} · {candidate.evidence}</p>
              </div>
              <div className="flex items-center gap-3">
                {candidate.detectedAt ? <time className="font-mono text-[10px] text-lyra-mutedDim">{new Date(candidate.detectedAt).toLocaleString()}</time> : null}
                <span className={`rounded border px-2 py-1 text-[10px] ${badgeTone(candidate.confidence)}`}>
                  {candidate.confidence === 'high' ? t('rfAttackTriage.highConfidence') : t('rfAttackTriage.mediumConfidence')}
                </span>
              </div>
            </article>
          ))}
        </div>
      )}

      <p className="text-[10px] text-lyra-mutedDim">{t('rfAttackTriage.disclaimer')}</p>
    </section>
  )
}
