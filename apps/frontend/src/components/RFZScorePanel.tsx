import { useMemo } from 'react'
import { useI18n } from '../lib/I18nContext'
import type { RFAnomaly } from '../lib/apiTypes'

type Band = 'watch' | 'high' | 'critical'

function bandFor(z: number): Band {
  const abs = Math.abs(z)
  if (abs >= 5) return 'critical'
  if (abs >= 4) return 'high'
  return 'watch'
}

function tone(band: Band) {
  if (band === 'critical') return 'border-lyra-bad/40 bg-lyra-bad/10 text-lyra-bad'
  if (band === 'high') return 'border-lyra-warn/40 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-border bg-lyra-bgSoft/40 text-lyra-muted'
}

export default function RFZScorePanel({ anomalies = [] }: { anomalies?: RFAnomaly[] }) {
  const { t } = useI18n()
  const rows = useMemo(() => [...anomalies]
    .filter((item) => Number.isFinite(item.z_score))
    .sort((a, b) => Math.abs(b.z_score) - Math.abs(a.z_score))
    .slice(0, 8), [anomalies])

  const summary = useMemo(() => {
    const values = anomalies.map((item) => Math.abs(item.z_score)).filter(Number.isFinite)
    return {
      max: values.length ? Math.max(...values) : 0,
      high: values.filter((value) => value >= 4).length,
      critical: values.filter((value) => value >= 5).length,
    }
  }, [anomalies])

  return (
    <section aria-labelledby="rf-zscore-title" className="grid gap-4">
      <div>
        <h3 id="rf-zscore-title" className="text-sm font-medium text-lyra-text">{t('rfZScore.title')}</h3>
        <p className="mt-1 text-xs text-lyra-muted">{t('rfZScore.subtitle')}</p>
      </div>

      {!rows.length ? (
        <div className="rounded border border-dashed border-lyra-border px-5 py-8 text-center">
          <p className="text-sm text-lyra-muted">{t('rfZScore.empty')}</p>
          <p className="mt-1 text-xs text-lyra-mutedDim">{t('rfZScore.emptyHint')}</p>
        </div>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded border border-lyra-borderSoft p-3"><p className="text-[10px] tracking-wider text-lyra-mutedDim">{t('rfZScore.max')}</p><p className="mt-1 font-mono text-lg text-lyra-text">{summary.max.toFixed(1)}σ</p></div>
            <div className="rounded border border-lyra-borderSoft p-3"><p className="text-[10px] tracking-wider text-lyra-mutedDim">{t('rfZScore.high')}</p><p className="mt-1 font-mono text-lg text-lyra-warn">{summary.high}</p></div>
            <div className="rounded border border-lyra-borderSoft p-3"><p className="text-[10px] tracking-wider text-lyra-mutedDim">{t('rfZScore.critical')}</p><p className="mt-1 font-mono text-lg text-lyra-bad">{summary.critical}</p></div>
          </div>
          <div className="overflow-x-auto rounded border border-lyra-borderSoft">
            <table className="w-full min-w-[720px] text-left text-sm">
              <thead className="bg-lyra-bgSoft/40"><tr>
                {['agent','metric','zscore','direction','detected'].map((key) => <th key={key} scope="col" className="px-3 py-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t(`rfZScore.${key}`)}</th>)}
              </tr></thead>
              <tbody>{rows.map((item) => { const band=bandFor(item.z_score); return (
                <tr key={item.id} className="border-t border-lyra-borderSoft/70">
                  <td className="px-3 py-3 font-mono text-xs text-lyra-goldBright">{item.agent_id}</td>
                  <td className="px-3 py-3 text-xs text-lyra-text">{item.metric}</td>
                  <td className="px-3 py-3"><span className={`rounded border px-2 py-1 font-mono text-xs ${tone(band)}`}>{item.z_score.toFixed(1)}σ</span></td>
                  <td className="px-3 py-3 text-xs text-lyra-muted">{item.z_score >= 0 ? t('rfZScore.above') : t('rfZScore.below')}</td>
                  <td className="px-3 py-3 font-mono text-xs text-lyra-mutedDim">{new Date(item.detected_at).toLocaleString()}</td>
                </tr>
              )})}</tbody>
            </table>
          </div>
        </>
      )}
      <p className="text-[10px] text-lyra-mutedDim">{t('rfZScore.disclaimer')}</p>
    </section>
  )
}
