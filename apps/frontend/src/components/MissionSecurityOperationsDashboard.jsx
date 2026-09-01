import { Activity, Clock3, Orbit, RadioTower, ShieldAlert, ShieldCheck, TriangleAlert } from 'lucide-react'
import StatCard from './StatCard.jsx'
import { useI18n } from '../lib/I18nContext'
import { calculateMissionRisk } from '../lib/risk'

const DAY_MS = 24 * 60 * 60 * 1000
const AGENT_ONLINE_WINDOW_MS = 5 * 60 * 1000
const INACTIVE_INCIDENT_STATUSES = new Set(['resolved', 'false_positive'])

function parseTime(value) {
  const time = Date.parse(value ?? '')
  return Number.isFinite(time) ? time : 0
}

function ageMs(value, now) {
  const time = parseTime(value)
  return time ? Math.max(0, now - time) : Number.POSITIVE_INFINITY
}

function hasPosition(satellite) {
  const position = satellite?.position
  return Number.isFinite(Number(position?.latitude_deg)) &&
    Number.isFinite(Number(position?.longitude_deg)) &&
    Number.isFinite(Number(position?.altitude_km))
}

function agentLastSeen(agent) {
  return agent?.last_seen_at || agent?.device?.reported_at || agent?.certificate?.last_seen_at
}

function agentPosture(agent, now) {
  if (agent?.status === 'quarantined' || agent?.status === 'revoked') return 'attention'
  const integrity = agent?.device?.capabilities?.integrity_status
  const compatibility = agent?.device?.compatibility
  if (integrity === 'changed' || compatibility === 'unsupported' || compatibility === 'degraded') return 'attention'
  const lastSeen = parseTime(agentLastSeen(agent))
  if (!lastSeen || now - lastSeen > AGENT_ONLINE_WINDOW_MS) return 'offline'
  return 'online'
}

function anomalySeverity(anomaly) {
  return Math.abs(Number(anomaly?.z_score) || 0) >= 5 ? 'high' : 'medium'
}

function severityRank(severity) {
  return ({ critical: 4, high: 3, medium: 2, low: 1 }[String(severity || '').toLowerCase()] ?? 0)
}

function severityClass(severity) {
  const normalized = String(severity || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'high') return 'border-lyra-bad/30 bg-lyra-bad/10 text-lyra-bad'
  if (normalized === 'medium') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  if (normalized === 'low') return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
  return 'border-lyra-border bg-lyra-bgSoft text-lyra-muted'
}

function stateAccent(state) {
  if (state === 'critical') return 'border-lyra-bad/30 bg-lyra-bad/10 text-lyra-bad'
  if (state === 'elevated') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
}

function formatMetric(anomaly) {
  if (String(anomaly?.metric || '').startsWith('frequency_hz')) return 'Frequency'
  if (String(anomaly?.metric || '').startsWith('power_dbm')) return 'Signal power'
  return anomaly?.metric || 'RF'
}

function buildActivity({ incidents, anomalies, alerts, t }) {
  const incidentItems = (incidents?.items ?? []).map((incident) => ({
    id: `incident-${incident.id}`,
    kind: 'incident',
    severity: incident.severity || 'unknown',
    time: incident.updated_at || incident.created_at,
    title: incident.title || incident.reference || t('missionSecurity.activityIncident'),
    detail: [incident.reference, incident.satellite_name || incident.agent_id].filter(Boolean).join(' · '),
    target: 'incidents',
  }))

  const anomalyItems = (anomalies ?? []).map((anomaly) => ({
    id: `anomaly-${anomaly.id}`,
    kind: 'rf',
    severity: anomalySeverity(anomaly),
    time: anomaly.detected_at,
    title: t('missionSecurity.rfAnomalyTitle', { agent: anomaly.agent_id || '—' }),
    detail: `${formatMetric(anomaly)} · ${(Number(anomaly.z_score) || 0).toFixed(1)}σ`,
    target: 'signal',
  }))

  const alertItems = (alerts?.items ?? []).filter((alert) => !alert.acknowledged).map((alert) => ({
    id: `alert-${alert.id}`,
    kind: 'cve',
    severity: alert.severity || 'unknown',
    time: alert.created_at,
    title: alert.cve_id || t('missionSecurity.activityVulnerability'),
    detail: [alert.asset_value, alert.matched_keyword].filter(Boolean).join(' · '),
    target: 'intel',
  }))

  return [...incidentItems, ...anomalyItems, ...alertItems]
    .sort((a, b) => parseTime(b.time) - parseTime(a.time) || severityRank(b.severity) - severityRank(a.severity))
    .slice(0, 7)
}

function QuickAction({ label, onClick }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded border border-lyra-border bg-lyra-bg/25 px-3 py-2 text-xs text-lyra-muted transition hover:border-lyra-gold/40 hover:text-lyra-goldBright"
    >
      {label}
    </button>
  )
}

export default function MissionSecurityOperationsDashboard({
  satellites = [],
  incidents = null,
  anomalies = null,
  agentIdentities = null,
  assetAlerts = null,
  live = false,
  backendStatus = 'checking',
  access = {},
  onNavigate = () => {},
  now = Date.now(),
}) {
  const { t } = useI18n()

  const incidentItems = incidents?.items ?? []
  const activeIncidents = incidentItems.filter((incident) => !INACTIVE_INCIDENT_STATUSES.has(incident.status))
  const criticalIncidents = activeIncidents.filter((incident) => incident.severity === 'critical').length
  const highIncidents = activeIncidents.filter((incident) => incident.severity === 'high').length
  const recentAnomalies = (anomalies ?? []).filter((anomaly) => ageMs(anomaly.detected_at, now) <= DAY_MS)
  const severeAnomalies = recentAnomalies.filter((anomaly) => severityRank(anomalySeverity(anomaly)) >= severityRank('high')).length
  const agents = agentIdentities ?? []
  const agentStates = agents.map((agent) => agentPosture(agent, now))
  const onlineAgents = agentStates.filter((state) => state === 'online').length
  const attentionAgents = agentStates.filter((state) => state === 'attention' || state === 'offline').length
  const unacknowledgedAlerts = (assetAlerts?.items ?? []).filter((alert) => !alert.acknowledged)
  const criticalAlerts = unacknowledgedAlerts.filter((alert) => ['critical', 'high'].includes(String(alert.severity).toLowerCase())).length
  const staleSatellites = satellites.filter((satellite) => ageMs(satellite.updated_at, now) > DAY_MS)
  const missingPositions = satellites.filter((satellite) => !hasPosition(satellite))

  const posture = backendStatus === 'offline' || criticalIncidents > 0
    ? 'critical'
    : highIncidents > 0 || severeAnomalies > 0 || criticalAlerts > 0 || attentionAgents > 0 || staleSatellites.length > 0
      ? 'elevated'
      : 'nominal'

  const queue = [
    ...(access.incidents ? activeIncidents
      .filter((incident) => ['critical', 'high'].includes(incident.severity))
      .map((incident) => ({ id: `incident-${incident.id}`, label: incident.title || incident.reference, detail: t('missionSecurity.queueIncident', { severity: String(incident.severity || '').toUpperCase() }), severity: incident.severity, target: 'incidents' })) : []),
    ...(access.alerts ? unacknowledgedAlerts
      .filter((alert) => ['critical', 'high'].includes(String(alert.severity).toLowerCase()))
      .map((alert) => ({ id: `alert-${alert.id}`, label: alert.cve_id, detail: t('missionSecurity.queueVulnerability', { asset: alert.asset_value || '—' }), severity: alert.severity, target: 'intel' })) : []),
    ...(access.agents ? agents
      .map((agent) => ({ agent, state: agentPosture(agent, now) }))
      .filter(({ state }) => state !== 'online')
      .map(({ agent, state }) => ({ id: `agent-${agent.agent_id}`, label: agent.agent_id, detail: t(`missionSecurity.queueAgent.${state}`), severity: state === 'attention' ? 'high' : 'medium', target: 'connectivity' })) : []),
    ...staleSatellites.map((satellite) => ({ id: `tle-${satellite.id}`, label: satellite.name, detail: t('missionSecurity.queueStaleTle'), severity: 'medium', target: 'orbital' })),
    ...missingPositions
      .filter((satellite) => !staleSatellites.some((stale) => stale.id === satellite.id))
      .map((satellite) => ({ id: `position-${satellite.id}`, label: satellite.name, detail: t('missionSecurity.queueMissingPosition'), severity: 'medium', target: 'orbital' })),
  ].sort((a, b) => severityRank(b.severity) - severityRank(a.severity)).slice(0, 6)

  const activity = buildActivity({
    incidents: access.incidents ? incidents : null,
    anomalies: access.rf ? anomalies : null,
    alerts: access.alerts ? assetAlerts : null,
    t,
  })

  const dataSources = [access.incidents, access.rf, access.agents, access.alerts].filter(Boolean).length
  const missionRisk = calculateMissionRisk({
    satellites,
    incidents: access.incidents ? incidentItems : [],
    anomalies: access.rf ? (anomalies ?? []) : [],
    agents: access.agents ? agents : [],
    alerts: access.alerts ? (assetAlerts?.items ?? []) : [],
    backendStatus,
    now,
  })

  return (
    <section aria-labelledby="mission-security-heading" className="grid gap-5">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('missionSecurity.eyebrow')}</p>
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <h1 id="mission-security-heading" className="font-display text-2xl font-medium text-lyra-text">{t('missionSecurity.title')}</h1>
            <span className={`inline-flex rounded-full border px-2.5 py-1 font-mono text-[10px] tracking-[0.14em] ${stateAccent(posture)}`}>
              {t(`missionSecurity.posture.${posture}`)}
            </span>
          </div>
          <p className="mt-1 max-w-3xl text-sm text-lyra-muted">{t('missionSecurity.subtitle')}</p>
          {dataSources < 4 && (
            <p className="mt-2 text-[11px] text-lyra-mutedDim">{t('missionSecurity.partialAccess', { count: dataSources, total: 4 })}</p>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {access.incidents && <QuickAction label={t('missionSecurity.openIncidents')} onClick={() => onNavigate('incidents')} />}
          {access.rf && <QuickAction label={t('missionSecurity.openSignal')} onClick={() => onNavigate('signal')} />}
          {access.agents && <QuickAction label={t('missionSecurity.openFleet')} onClick={() => onNavigate('connectivity')} />}
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-7">
        <StatCard icon={ShieldCheck} label={t('missionSecurity.metric.riskScore')} value={`${missionRisk.score}/100`} accent={missionRisk.level === 'critical' ? 'bad' : missionRisk.level === 'elevated' ? 'warn' : 'good'} />
        <StatCard icon={Orbit} label={t('missionSecurity.metric.satellites')} value={satellites.length} />
        <StatCard icon={ShieldAlert} label={t('missionSecurity.metric.criticalIncidents')} value={access.incidents ? criticalIncidents : '—'} accent={criticalIncidents ? 'bad' : 'good'} />
        <StatCard icon={TriangleAlert} label={t('missionSecurity.metric.rfAnomalies')} value={access.rf ? recentAnomalies.length : '—'} accent={severeAnomalies ? 'bad' : recentAnomalies.length ? 'warn' : 'good'} />
        <StatCard icon={RadioTower} label={t('missionSecurity.metric.agentsOnline')} value={access.agents ? `${onlineAgents}/${agents.length}` : '—'} accent={access.agents && agents.length > 0 ? (attentionAgents === 0 ? 'good' : 'warn') : 'neutral'} />
        <StatCard icon={Activity} label={t('missionSecurity.metric.assetAlerts')} value={access.alerts ? unacknowledgedAlerts.length : '—'} accent={criticalAlerts ? 'bad' : unacknowledgedAlerts.length ? 'warn' : 'good'} />
        <StatCard icon={live ? ShieldCheck : Clock3} label={t('missionSecurity.metric.orbitalFeed')} value={live ? t('missionSecurity.live') : t('missionSecurity.polling')} accent={live ? 'good' : 'neutral'} />
      </div>

      <div className="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-5">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">{t('missionSecurity.queueEyebrow')}</p>
              <h2 className="mt-1 font-display text-base font-medium text-lyra-text">{t('missionSecurity.queueTitle')}</h2>
            </div>
            <TriangleAlert className={`h-4 w-4 ${queue.length ? 'text-lyra-warn' : 'text-lyra-good'}`} aria-hidden="true" />
          </div>
          {queue.length === 0 ? (
            <div className="mt-4 rounded border border-lyra-good/20 bg-lyra-good/5 px-3 py-4">
              <p className="text-sm text-lyra-good">{t('missionSecurity.queueHealthy')}</p>
              <p className="mt-1 text-xs text-lyra-mutedDim">{t('missionSecurity.queueHealthyHint')}</p>
            </div>
          ) : (
            <ul className="mt-4 grid gap-2">
              {queue.map((item) => (
                <li key={item.id}>
                  <button type="button" onClick={() => onNavigate(item.target)} className="flex w-full items-center justify-between gap-3 rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2.5 text-left hover:border-lyra-gold/30">
                    <span className="min-w-0">
                      <span className="block truncate text-sm text-lyra-text">{item.label}</span>
                      <span className="mt-0.5 block truncate text-[11px] text-lyra-mutedDim">{item.detail}</span>
                    </span>
                    <span className={`shrink-0 rounded border px-2 py-0.5 font-mono text-[9px] tracking-wider ${severityClass(item.severity)}`}>{String(item.severity || 'info').toUpperCase()}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-5">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">{t('missionSecurity.activityEyebrow')}</p>
              <h2 className="mt-1 font-display text-base font-medium text-lyra-text">{t('missionSecurity.activityTitle')}</h2>
            </div>
            <Activity className="h-4 w-4 text-lyra-mutedDim" aria-hidden="true" />
          </div>
          {activity.length === 0 ? (
            <p className="mt-4 rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-4 text-sm text-lyra-muted">{t('missionSecurity.activityEmpty')}</p>
          ) : (
            <ol className="mt-4 divide-y divide-lyra-borderSoft">
              {activity.map((item) => (
                <li key={item.id}>
                  <button type="button" onClick={() => onNavigate(item.target)} className="grid w-full grid-cols-[auto_1fr_auto] items-center gap-3 py-2.5 text-left first:pt-0 last:pb-0">
                    <span className={`h-2 w-2 rounded-full ${severityRank(item.severity) >= 3 ? 'bg-lyra-bad' : severityRank(item.severity) === 2 ? 'bg-lyra-warn' : 'bg-lyra-good'}`} aria-hidden="true" />
                    <span className="min-w-0">
                      <span className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                        <span className="truncate text-sm text-lyra-text">{item.title}</span>
                        <span className="font-mono text-[9px] tracking-wider text-lyra-mutedDim">{t(`missionSecurity.activityKind.${item.kind}`)}</span>
                      </span>
                      {item.detail && <span className="mt-0.5 block truncate text-[11px] text-lyra-mutedDim">{item.detail}</span>}
                    </span>
                    <time className="shrink-0 font-mono text-[10px] text-lyra-muted" dateTime={item.time || undefined}>{item.time ? new Date(item.time).toLocaleString() : '—'}</time>
                  </button>
                </li>
              ))}
            </ol>
          )}
        </div>
      </div>
    </section>
  )
}
