import TableScroll from './TableScroll.jsx'

const METRIC_LABELS = {
  power_dbm: 'Signal power',
  frequency_hz: 'Frequency',
  power_dbm_trend: 'Signal power (trend)',
  frequency_hz_trend: 'Frequency (trend)',
}

function formatValue(metric, value) {
  if (metric.startsWith('frequency_hz')) return `${(value / 1e6).toFixed(3)} MHz`
  return `${value.toFixed(1)} dBm`
}

function severityTone(absZ) {
  if (absZ >= 5) return 'text-lyra-bad'
  return 'text-lyra-warn'
}

export default function AnomaliesPanel({ anomalies }) {
  if (!anomalies.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">No anomalies detected.</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">
          Each reading is checked two ways: a spot outlier against a robust median/MAD baseline,
          and a sustained-drift check against an EWMA trend baseline.
        </p>
      </div>
    )
  }

  return (
    <TableScroll>
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-lyra-borderSoft">
            <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Agent
            </th>
            <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Metric
            </th>
            <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Value
            </th>
            <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Baseline
            </th>
            <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Z-score
            </th>
            <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Detected
            </th>
          </tr>
        </thead>
        <tbody>
          {anomalies.map((a) => (
            <tr key={a.id} className="border-b border-lyra-borderSoft/60 last:border-0">
              <td className="py-2.5 font-mono text-xs text-lyra-goldBright">{a.agent_id}</td>
              <td className="py-2.5 text-xs text-lyra-text">{METRIC_LABELS[a.metric] ?? a.metric}</td>
              <td className="py-2.5 font-mono text-xs text-lyra-text">{formatValue(a.metric, a.value)}</td>
              <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">
                {formatValue(a.metric, a.baseline_mean)} ± {formatValue(a.metric, a.baseline_stddev)}
              </td>
              <td className={`py-2.5 font-mono text-xs ${severityTone(Math.abs(a.z_score))}`}>
                {a.z_score.toFixed(1)}σ
              </td>
              <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">
                {new Date(a.detected_at).toLocaleString()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </TableScroll>
  )
}
