import { useMemo } from 'react'
import { Cpu, Radio, Satellite, ShieldCheck, Thermometer, TimerReset } from 'lucide-react'
import { useI18n } from '../lib/I18nContext.jsx'
import TableScroll from './TableScroll.jsx'

function formatHz(value) {
  if (!value) return '—'
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(3)} GHz`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(value % 1_000_000 === 0 ? 0 : 3)} MHz`
  if (value >= 1_000) return `${(value / 1_000).toFixed(value % 1_000 === 0 ? 0 : 1)} kHz`
  return `${value} Hz`
}

function compatibilityClass(status) {
  if (status === 'compatible') return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
  if (status === 'degraded') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  if (status === 'stale') return 'border-lyra-border bg-lyra-bgSoft text-lyra-muted'
  return 'border-lyra-bad/30 bg-lyra-bad/10 text-lyra-bad'
}

function mergeAgents(agents, legacyKeys) {
  const merged = new Map()
  for (const agent of agents ?? []) merged.set(agent.agent_id, { ...agent, source: 'mtls' })
  for (const key of legacyKeys ?? []) {
    if (!merged.has(key.agent_id)) {
      merged.set(key.agent_id, {
        agent_id: key.agent_id,
        status: key.revoked ? 'revoked' : 'active',
        last_seen_at: key.last_used_at,
        device: key.device,
        source: 'legacy',
      })
    } else if (!merged.get(key.agent_id).device && key.device) {
      merged.set(key.agent_id, { ...merged.get(key.agent_id), device: key.device })
    }
  }
  return [...merged.values()].sort((a, b) => a.agent_id.localeCompare(b.agent_id))
}

export default function AgentHardwarePanel({ agents = [], legacyKeys = [] }) {
  const { t } = useI18n()
  const rows = useMemo(() => mergeAgents(agents, legacyKeys), [agents, legacyKeys])
  const reported = rows.filter((item) => item.device)
  const counts = reported.reduce((acc, item) => {
    const key = item.device?.compatibility || 'unsupported'
    acc[key] = (acc[key] || 0) + 1
    return acc
  }, {})

  if (!rows.length) return <p className="text-sm text-lyra-muted">{t('connectivity.hardwareEmpty')}</p>

  return (
    <div className="grid gap-5">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-label={t('connectivity.hardwareSummary')}>
        {['compatible', 'degraded', 'unsupported', 'stale'].map((status) => (
          <div key={status} className="rounded-lg border border-lyra-border bg-lyra-panel/30 px-4 py-3">
            <p className="font-mono text-[10px] tracking-wider text-lyra-mutedDim">{t(`connectivity.compatibility.${status}`)}</p>
            <p className="mt-1 font-display text-2xl text-lyra-text">{counts[status] || 0}</p>
          </div>
        ))}
      </div>

      <TableScroll>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-lyra-borderSoft">
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('connectivity.agent')}</th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('connectivity.device')}</th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('connectivity.hardwareCapability')}</th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('connectivity.hardwareCompatibility')}</th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('connectivity.reportedAt')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((agent) => {
              const device = agent.device
              const caps = device?.capabilities || {}
              const status = device?.compatibility || 'unsupported'
              return (
                <tr key={agent.agent_id} className="border-b border-lyra-borderSoft/60 align-top last:border-0">
                  <td className="py-3 pr-5">
                    <div className="font-mono text-xs text-lyra-goldBright">{agent.agent_id}</div>
                    <div className="mt-1 font-mono text-[10px] text-lyra-mutedDim">{agent.source === 'legacy' ? t('connectivity.legacyAgentKey') : 'mTLS'}</div>
                  </td>
                  <td className="py-3 pr-5">
                    {device ? (
                      <div>
                        <div className="font-mono text-xs text-lyra-text">{device.hardware || device.product || device.driver}</div>
                        <div className="mt-1 text-xs text-lyra-muted">{[device.manufacturer, device.product].filter(Boolean).join(' · ') || '—'}</div>
                        <div className="mt-1 font-mono text-[10px] text-lyra-mutedDim">{device.driver}{caps.driver_version ? ` ${caps.driver_version}` : ''}{device.serial ? ` · ${device.serial}` : ''}</div>
                      </div>
                    ) : <span className="text-xs text-lyra-mutedDim">{t('connectivity.noDeviceReported')}</span>}
                  </td>
                  <td className="py-3 pr-5">
                    {device ? (
                      <details className="min-w-[16rem]">
                        <summary className="cursor-pointer font-mono text-xs text-lyra-goldBright outline-none focus-visible:underline">{t('connectivity.hardwareDetails')}</summary>
                        <div className="mt-2 grid gap-1.5 text-xs text-lyra-muted">
                          <span className="flex items-center gap-1.5"><Cpu className="h-3.5 w-3.5" />{caps.architecture || '—'} / {caps.os || '—'} · {caps.channels || 0} ch</span>
                          <span className="flex items-center gap-1.5"><Radio className="h-3.5 w-3.5" />{formatHz(caps.frequency_min_hz)} – {formatHz(caps.frequency_max_hz)}</span>
                          <span>{t('connectivity.sampleRates')}: {(caps.sample_rates_hz || []).map(formatHz).join(', ') || '—'}</span>
                          <span>{t('connectivity.bandwidths')}: {(caps.bandwidths_hz || []).map(formatHz).join(', ') || '—'}</span>
                          <span>{t('connectivity.gains')}: {(caps.gain_elements || []).join(', ') || '—'}</span>
                          <span className="flex items-center gap-1.5"><Satellite className="h-3.5 w-3.5" />GPS {caps.gps ? '✓' : '—'} · PPS {caps.pps ? '✓' : '—'} · RX {device.rx ? '✓' : '—'} · TX {device.tx ? '✓' : '—'}</span>
                          {caps.temperature_c != null && <span className="flex items-center gap-1.5"><Thermometer className="h-3.5 w-3.5" />{caps.temperature_c.toFixed(1)} °C</span>}
                          <span className="flex items-center gap-1.5"><TimerReset className="h-3.5 w-3.5" />{t('connectivity.sampleDrops')}: {caps.sample_drops ?? 0}</span>
                          <span className="flex items-center gap-1.5"><ShieldCheck className="h-3.5 w-3.5" />{t('connectivity.edgeIntegrity')}: {caps.integrity_status ? t(`connectivity.integrity.${caps.integrity_status}`) : '—'}</span>
                          {caps.integrity_manifest_sha256 && <span className="break-all font-mono text-[10px]">SHA-256 {caps.integrity_manifest_sha256}</span>}
                          {caps.integrity_file_count != null && <span>{t('connectivity.integrityFiles')}: {caps.integrity_file_count}</span>}
                        </div>
                      </details>
                    ) : '—'}
                  </td>
                  <td className="py-3 pr-5">
                    {device ? (
                      <div className="max-w-72">
                        <span className={`inline-flex rounded-full border px-2 py-0.5 font-mono text-[10px] tracking-wider ${compatibilityClass(status)}`}>{t(`connectivity.compatibility.${status}`)}</span>
                        <p className="mt-1.5 text-xs text-lyra-muted">{device.compatibility_reason}</p>
                      </div>
                    ) : '—'}
                  </td>
                  <td className="py-3 font-mono text-xs text-lyra-mutedDim">{device?.reported_at ? new Date(device.reported_at).toLocaleString() : '—'}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </TableScroll>
      <p className="text-xs text-lyra-mutedDim">{t('connectivity.hardwareValidationNote')}</p>
    </div>
  )
}
