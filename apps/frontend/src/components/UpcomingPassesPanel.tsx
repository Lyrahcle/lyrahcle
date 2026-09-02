import { Clock3, RadioTower, Satellite, ShieldAlert } from 'lucide-react'
import type { SatellitePass } from '../lib/apiTypes'
import { useI18n } from '../lib/I18nContext'

type PassState = 'nominal' | 'rf_attention' | 'incident_active' | 'agent_offline'

interface UpcomingPassesPanelProps {
  passes: SatellitePass[] | null
  stationId?: string
  agentStatus?: string
  agentLastSeenAt?: string
  activeIncidentCount?: number
  anomalyTimes?: string[]
  now?: number
  access?: Record<string, boolean>
  onNavigate?: (tab: string) => void
}

const TEN_MINUTES = 10 * 60 * 1000
const FIFTEEN_MINUTES = 15 * 60 * 1000

function time(value?: string): number {
  const parsed = Date.parse(value || '')
  return Number.isFinite(parsed) ? parsed : 0
}

function formatTime(value?: string): string {
  const parsed = time(value)
  if (!parsed) return '—'
  return new Date(parsed).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function stateForPass({
  pass,
  activeIncidentCount,
  anomalyTimes,
  agentStatus,
  agentLastSeenAt,
  now,
}: {
  pass: SatellitePass
  activeIncidentCount: number
  anomalyTimes: string[]
  agentStatus?: string
  agentLastSeenAt?: string
  now: number
}): PassState {
  if (activeIncidentCount > 0) return 'incident_active'
  const lastSeen = time(agentLastSeenAt)
  if (agentStatus === 'quarantined' || agentStatus === 'revoked' || (lastSeen > 0 && now - lastSeen > TEN_MINUTES)) return 'agent_offline'
  const aos = time(pass.aos)
  const los = time(pass.los)
  if (anomalyTimes.some((value) => {
    const detected = time(value)
    return detected >= aos - FIFTEEN_MINUTES && detected <= los + FIFTEEN_MINUTES
  })) return 'rf_attention'
  return 'nominal'
}

function stateClasses(state: PassState): string {
  if (state === 'incident_active') return 'border-lyra-bad/30 bg-lyra-bad/10 text-lyra-bad'
  if (state === 'agent_offline' || state === 'rf_attention') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
}

export default function UpcomingPassesPanel({
  passes,
  stationId = '',
  agentStatus,
  agentLastSeenAt,
  activeIncidentCount = 0,
  anomalyTimes = [],
  now = Date.now(),
  access = {},
  onNavigate = () => {},
}: UpcomingPassesPanelProps) {
  const { t } = useI18n()
  const upcoming = (passes ?? []).filter((pass) => time(pass.los) >= now).slice(0, 3)

  return <section aria-labelledby="upcoming-passes-heading" className="rounded-xl border border-lyra-border bg-lyra-panel/40 p-5">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div>
        <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('missionControl.passesEyebrow')}</p>
        <h2 id="upcoming-passes-heading" className="mt-1 font-display text-xl text-lyra-text">{t('missionControl.upcomingPasses')}</h2>
        <p className="mt-1 text-xs text-lyra-muted">{stationId ? t('missionControl.passStation', { station: stationId }) : t('missionControl.passStationUnavailable')}</p>
      </div>
      <Satellite className="h-5 w-5 text-lyra-gold" />
    </div>

    {passes === null ? <p className="mt-4 text-sm text-lyra-muted">{t('missionControl.loadingPasses')}</p> : upcoming.length === 0 ? <p className="mt-4 text-sm text-lyra-muted">{t('missionControl.noUpcomingPasses')}</p> : <div className="mt-4 overflow-x-auto">
      <table className="w-full min-w-[680px] text-left">
        <thead><tr className="border-b border-lyra-borderSoft text-[10px] tracking-[0.12em] text-lyra-mutedDim"><th className="pb-2 font-normal">AOS</th><th className="pb-2 font-normal">{t('missionControl.peak')}</th><th className="pb-2 font-normal">LOS</th><th className="pb-2 font-normal">{t('missionControl.maxElevation')}</th><th className="pb-2 font-normal">{t('missionControl.passStatus')}</th><th className="pb-2 font-normal text-right">{t('missionControl.actions')}</th></tr></thead>
        <tbody>{upcoming.map((pass) => {
          const state = stateForPass({ pass, activeIncidentCount, anomalyTimes, agentStatus, agentLastSeenAt, now })
          return <tr key={`${pass.aos}-${pass.los}`} className="border-b border-lyra-borderSoft/70 last:border-b-0">
            <td className="py-3 font-mono text-xs text-lyra-text"><span className="inline-flex items-center gap-1.5"><Clock3 className="h-3.5 w-3.5 text-lyra-mutedDim" />{formatTime(pass.aos)}</span></td>
            <td className="py-3 font-mono text-xs text-lyra-text">{formatTime(pass.peak_at)}</td>
            <td className="py-3 font-mono text-xs text-lyra-text">{formatTime(pass.los)}</td>
            <td className="py-3 font-mono text-xs text-lyra-text">{Number.isFinite(pass.max_elevation_deg) ? `${pass.max_elevation_deg.toFixed(1)}°` : '—'}</td>
            <td className="py-3"><span className={`inline-flex rounded-full border px-2 py-1 font-mono text-[10px] ${stateClasses(state)}`}>{t(`missionControl.passState.${state}`)}</span></td>
            <td className="py-3"><div className="flex justify-end gap-2">{access.rf && <button type="button" onClick={() => onNavigate('signal')} aria-label={t('missionControl.openRf')} className="rounded border border-lyra-border p-2 text-lyra-muted hover:text-lyra-goldBright"><RadioTower className="h-3.5 w-3.5" /></button>}{access.incidents && <button type="button" onClick={() => onNavigate('incidents')} aria-label={t('missionControl.openIncidentWorkspace')} className="rounded border border-lyra-border p-2 text-lyra-muted hover:text-lyra-goldBright"><ShieldAlert className="h-3.5 w-3.5" /></button>}</div></td>
          </tr>
        })}</tbody>
      </table>
    </div>}
  </section>
}
