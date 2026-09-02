import { useMemo } from 'react'
import { useI18n } from '../lib/I18nContext'
import type { RFDopplerComparison } from '../lib/apiTypes'

function fmtHz(value: number) {
  const abs = Math.abs(value)
  if (abs >= 1e6) return `${(value / 1e6).toFixed(6)} MHz`
  if (abs >= 1e3) return `${(value / 1e3).toFixed(2)} kHz`
  return `${value.toFixed(1)} Hz`
}

function signedHz(value: number) {
  return `${value >= 0 ? '+' : ''}${fmtHz(value)}`
}

function ratio(item: RFDopplerComparison) {
  if (!Number.isFinite(item.tolerance_hz) || item.tolerance_hz <= 0) return 0
  return Math.abs(item.residual_hz) / item.tolerance_hz
}

export default function RFDopplerPanel({ comparisons = [] }: { comparisons?: RFDopplerComparison[] }) {
  const { t } = useI18n()
  const rows = useMemo(
    () => [...comparisons].sort((a, b) => Math.abs(b.residual_hz) - Math.abs(a.residual_hz)).slice(0, 8),
    [comparisons],
  )

  return (
    <section aria-labelledby="rf-doppler-title" className="grid gap-4">
      <div>
        <h3 id="rf-doppler-title" className="text-sm font-medium text-lyra-text">{t('rfDoppler.title')}</h3>
        <p className="mt-1 text-xs text-lyra-muted">{t('rfDoppler.subtitle')}</p>
      </div>

      {!rows.length ? (
        <div className="rounded border border-dashed border-lyra-border px-5 py-8 text-center">
          <p className="text-sm text-lyra-muted">{t('rfDoppler.empty')}</p>
          <p className="mt-1 text-xs text-lyra-mutedDim">{t('rfDoppler.emptyHint')}</p>
        </div>
      ) : (
        <div className="overflow-x-auto rounded border border-lyra-borderSoft">
          <table className="w-full min-w-[900px] text-left text-sm">
            <thead className="bg-lyra-bgSoft/40">
              <tr>
                {['satellite','agent','observed','expected','doppler','residual','status'].map((key) => (
                  <th key={key} scope="col" className="px-3 py-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                    {t(`rfDoppler.${key}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((item) => {
                const outside = ratio(item) > 1
                return (
                  <tr key={item.anomaly_id} className="border-t border-lyra-borderSoft/70">
                    <td className="px-3 py-3">
                      <p className="text-xs text-lyra-text">{item.satellite_name}</p>
                      <p className="font-mono text-[10px] text-lyra-mutedDim">{new Date(item.detected_at).toLocaleString()}</p>
                    </td>
                    <td className="px-3 py-3 font-mono text-xs text-lyra-goldBright">{item.agent_id}</td>
                    <td className="px-3 py-3 font-mono text-xs text-lyra-text">{fmtHz(item.observed_frequency_hz)}</td>
                    <td className="px-3 py-3 font-mono text-xs text-lyra-text">{fmtHz(item.expected_frequency_hz)}</td>
                    <td className="px-3 py-3 font-mono text-xs text-lyra-muted">{signedHz(item.expected_doppler_hz)}</td>
                    <td className="px-3 py-3 font-mono text-xs text-lyra-text">{signedHz(item.residual_hz)}</td>
                    <td className="px-3 py-3">
                      <span className={`rounded border px-2 py-1 text-[10px] ${outside ? 'border-lyra-bad/40 bg-lyra-bad/10 text-lyra-bad' : 'border-lyra-good/40 bg-lyra-good/10 text-lyra-good'}`}>
                        {outside ? t('rfDoppler.outsideTolerance') : t('rfDoppler.withinTolerance')}
                      </span>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
      <p className="text-[10px] text-lyra-mutedDim">{t('rfDoppler.disclaimer')}</p>
    </section>
  )
}
