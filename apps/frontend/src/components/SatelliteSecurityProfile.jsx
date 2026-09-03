import { useEffect, useMemo, useState } from 'react'
import { Activity, ArrowLeft, Clock3, Orbit, RadioTower, Satellite, ShieldAlert, ShieldCheck, TriangleAlert } from 'lucide-react'
import { fetchIncidents, fetchOrbitalEvents, fetchSatelliteExternalSources } from '../lib/api'
import { useI18n } from '../lib/I18nContext'
import { calculateSatelliteRisk, riskSeverityRank } from '../lib/risk'
import EvidenceBadge from './EvidenceBadge'

const DAY_MS = 24 * 60 * 60 * 1000
const INACTIVE_INCIDENT_STATUSES = new Set(['resolved', 'false_positive'])

function parseTime(value) {
  const parsed = Date.parse(value ?? '')
  return Number.isFinite(parsed) ? parsed : 0
}

function tone(value) {
  const normalized = String(value || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'high' || normalized === 'unusable') return 'border-lyra-bad/30 bg-lyra-bad/10 text-lyra-bad'
  if (normalized === 'medium' || normalized === 'aging' || normalized === 'stale') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
}

function metricLabel(metric) {
  if (String(metric || '').startsWith('frequency_hz')) return 'Frequency'
  if (String(metric || '').startsWith('power_dbm')) return 'Signal power'
  return metric || 'RF'
}

function formatFrequency(value) {
  const hz = Number(value)
  if (!Number.isFinite(hz) || hz <= 0) return '—'
  return `${(hz / 1_000_000).toFixed(4)} MHz`
}

function SatelliteSources({ intelligence, loading, error, t }) {
  if (loading) return <div className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-5 text-sm text-lyra-muted">{t('satelliteSources.loading')}</div>
  if (error) return <div className="rounded-lg border border-lyra-warn/30 bg-lyra-warn/5 p-5 text-sm text-lyra-warn">{t('satelliteSources.unavailable')}: {error}</div>
  if (!intelligence) return null
  return (
    <section aria-labelledby="satellite-real-sources-title" className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div><p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('satelliteSources.eyebrow')}</p><h2 id="satellite-real-sources-title" className="mt-1 font-display text-lg text-lyra-text">{t('satelliteSources.title')}</h2></div>
        <div className="flex items-center gap-2"><EvidenceBadge kind="external" /><span className={`rounded-full border px-2 py-1 font-mono text-[10px] ${tone(intelligence.overall_quality)}`}>{intelligence.overall_quality}</span></div>
      </div>
      <p className="mt-2 text-xs text-lyra-muted">{t('satelliteSources.contextNote')}</p>
      <div className="mt-4 grid gap-3 md:grid-cols-3">
        {(intelligence.sources || []).map((source) => <div key={source.name} className="rounded border border-lyra-borderSoft bg-lyra-bg/25 p-3"><div className="flex items-center justify-between gap-2"><p className="text-xs text-lyra-text">{source.name}</p><span className={`h-2 w-2 rounded-full ${source.available ? 'bg-lyra-good' : 'bg-lyra-bad'}`} /></div><p className="mt-2 font-mono text-[10px] text-lyra-mutedDim">{source.record_count} · {source.freshness}</p><p className="mt-1 text-[10px] text-lyra-mutedDim">{new Date(source.fetched_at).toLocaleString()}</p></div>)}
      </div>
      <div className="mt-4 grid gap-4 lg:grid-cols-2">
        <div className="rounded border border-lyra-borderSoft p-4"><p className="text-[10px] tracking-[0.14em] text-lyra-mutedDim">{t('satelliteSources.metadata')}</p><dl className="mt-3 grid gap-2 text-xs"><div className="flex justify-between gap-3"><dt className="text-lyra-muted">NORAD</dt><dd className="font-mono text-lyra-text">{intelligence.norad_id}</dd></div><div className="flex justify-between gap-3"><dt className="text-lyra-muted">SatNOGS</dt><dd className="font-mono text-lyra-text">{intelligence.metadata?.satnogs_id || '—'}</dd></div><div className="flex justify-between gap-3"><dt className="text-lyra-muted">{t('satelliteSources.status')}</dt><dd className="font-mono text-lyra-text">{intelligence.metadata?.status || '—'}</dd></div><div className="flex justify-between gap-3"><dt className="text-lyra-muted">{t('satelliteSources.lastReception')}</dt><dd className="font-mono text-lyra-text">{intelligence.last_reception ? new Date(intelligence.last_reception).toLocaleString() : '—'}</dd></div></dl></div>
        <div className="rounded border border-lyra-borderSoft p-4"><p className="text-[10px] tracking-[0.14em] text-lyra-mutedDim">{t('satelliteSources.transmitters')}</p><div className="mt-3 grid gap-2">{(intelligence.transmitters || []).length ? intelligence.transmitters.slice(0, 8).map((tx) => <div key={tx.uuid} className="rounded bg-lyra-bg/30 px-3 py-2"><div className="flex justify-between gap-3"><span className="truncate text-xs text-lyra-text">{tx.description || tx.uuid}</span><span className="font-mono text-[10px] text-lyra-mutedDim">{tx.mode || tx.status || '—'}</span></div><p className="mt-1 font-mono text-[10px] text-lyra-goldBright">↓ {formatFrequency(tx.downlink_low_hz)}{tx.uplink_low_hz ? ` · ↑ ${formatFrequency(tx.uplink_low_hz)}` : ''}</p></div>) : <p className="text-xs text-lyra-mutedDim">{t('satelliteSources.noTransmitters')}</p>}</div></div>
      </div>
      <div className="mt-4 rounded border border-lyra-borderSoft p-4"><p className="text-[10px] tracking-[0.14em] text-lyra-mutedDim">{t('satelliteSources.latestObservations')}</p><div className="mt-3 grid gap-2 md:grid-cols-2">{(intelligence.observations || []).length ? intelligence.observations.slice(0, 6).map((observation) => <a key={observation.id} href={`https://network.satnogs.org/observations/${observation.id}/`} target="_blank" rel="noopener noreferrer" className="rounded bg-lyra-bg/30 px-3 py-2 hover:bg-lyra-bgSoft"><div className="flex justify-between gap-3"><span className="font-mono text-xs text-lyra-goldBright">#{observation.id}</span><span className="text-[10px] text-lyra-mutedDim">{observation.status || '—'}</span></div><p className="mt-1 text-[10px] text-lyra-mutedDim">{observation.end ? new Date(observation.end).toLocaleString() : '—'}{observation.station ? ` · ${observation.station}` : ''}</p></a>) : <p className="text-xs text-lyra-mutedDim">{t('satelliteSources.noObservations')}</p>}</div></div>
    </section>
  )
}

function RiskGauge({ score, level, factors, t }) {
  return (
    <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('satelliteProfile.riskEyebrow')}</p>
          <p className="mt-2 font-mono text-4xl text-lyra-text">{score}<span className="text-base text-lyra-mutedDim">/100</span></p>
        </div>
        <span className={`rounded-full border px-2.5 py-1 font-mono text-[10px] tracking-[0.14em] ${tone(level)}`}>{t(`satelliteProfile.risk.${level}`)}</span>
      </div>
      <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-lyra-bgSoft" aria-label={t('satelliteProfile.riskScore', { score })}>
        <div className="h-full bg-lyra-goldBright" style={{ width: `${score}%` }} />
      </div>
      <div className="mt-4 grid gap-2">
        {factors.length ? factors.map((factor) => (
          <div key={factor.key} className="flex items-center justify-between gap-4 text-xs">
            <span className="text-lyra-muted">{t(`satelliteProfile.riskFactor.${factor.key}`)}</span>
            <span className="font-mono text-lyra-warn">+{factor.points}</span>
          </div>
        )) : <p className="text-xs text-lyra-good">{t('satelliteProfile.noRiskFactors')}</p>}
      </div>
    </div>
  )
}

function TimelineItem({ icon: Icon, label, detail, time, severity }) {
  return (
    <li className="flex gap-3 border-b border-lyra-borderSoft/70 py-3 last:border-0">
      <span className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border ${tone(severity)}`}><Icon className="h-3.5 w-3.5" /></span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <p className="truncate text-sm text-lyra-text">{label}</p>
          <time className="font-mono text-[10px] text-lyra-mutedDim">{time ? new Date(time).toLocaleString() : '—'}</time>
        </div>
        {detail && <p className="mt-1 text-xs text-lyra-muted">{detail}</p>}
      </div>
    </li>
  )
}

export default function SatelliteSecurityProfile({
  satellite,
  incidents = null,
  anomalies = null,
  agentIdentities = null,
  assetAlerts = null,
  access = {},
  onBack = () => {},
  onNavigate = () => {},
  now = Date.now(),
}) {
  const { t } = useI18n()
  const [orbitalEvents, setOrbitalEvents] = useState(null)
  const [profileIncidents, setProfileIncidents] = useState(null)
  const [eventsError, setEventsError] = useState('')
  const [sourceIntelligence, setSourceIntelligence] = useState(null)
  const [sourcesError, setSourcesError] = useState('')

  useEffect(() => {
    let cancelled = false
    setOrbitalEvents(null)
    setEventsError('')
    fetchOrbitalEvents(satellite.id, 168, 50)
      .then((data) => { if (!cancelled) setOrbitalEvents(Array.isArray(data) ? data : []) })
      .catch((error) => {
        if (!cancelled) {
          setOrbitalEvents([])
          setEventsError(error?.message || t('satelliteProfile.orbitalEventsUnavailable'))
        }
      })
    return () => { cancelled = true }
  }, [satellite.id, t])

  useEffect(() => {
    let cancelled = false
    setSourceIntelligence(null)
    setSourcesError('')
    fetchSatelliteExternalSources(satellite.id)
      .then((data) => { if (!cancelled) setSourceIntelligence(data) })
      .catch((error) => { if (!cancelled) setSourcesError(error?.message || t('satelliteSources.unavailable')) })
    return () => { cancelled = true }
  }, [satellite.id, t])

  useEffect(() => {
    let cancelled = false
    if (!access.incidents) {
      setProfileIncidents(null)
      return undefined
    }
    fetchIncidents(50, 0, { satelliteId: satellite.id })
      .then((page) => { if (!cancelled) setProfileIncidents(page) })
      .catch(() => { if (!cancelled) setProfileIncidents(null) })
    return () => { cancelled = true }
  }, [access.incidents, satellite.id])

  const satelliteIncidents = useMemo(() => access.incidents
    ? ((profileIncidents?.items ?? incidents?.items ?? []).filter((item) => item.satellite_id === satellite.id))
    : [], [access.incidents, incidents, profileIncidents, satellite.id])
  const satelliteAnomalies = useMemo(() => access.rf
    ? (anomalies ?? []).filter((item) => item.satellite_id === satellite.id)
    : [], [access.rf, anomalies, satellite.id])
  const relatedAgentIds = useMemo(() => new Set([
    ...satelliteIncidents.map((item) => item.agent_id).filter(Boolean),
    ...satelliteAnomalies.map((item) => item.agent_id).filter(Boolean),
  ]), [satelliteIncidents, satelliteAnomalies])
  const relatedAgents = access.agents ? (agentIdentities ?? []).filter((item) => relatedAgentIds.has(item.agent_id)) : []
  const exposureAlerts = useMemo(() => {
    if (!access.alerts) return []
    const identities = new Set([String(satellite.name || '').toLowerCase(), String(satellite.norad_id ?? '').toLowerCase()])
    return (assetAlerts?.items ?? []).filter((item) => identities.has(String(item.asset_value || '').toLowerCase()))
  }, [access.alerts, assetAlerts, satellite.name, satellite.norad_id])
  const events = orbitalEvents ?? []
  const risk = calculateSatelliteRisk({ satellite, incidents: satelliteIncidents, anomalies: satelliteAnomalies, orbitalEvents: events, exposureAlerts, now })
  const openIncidents = satelliteIncidents.filter((item) => !INACTIVE_INCIDENT_STATUSES.has(item.status))
  const recentAnomalies = satelliteAnomalies.filter((item) => now - parseTime(item.detected_at) <= DAY_MS)

  const timeline = [
    ...satelliteIncidents.map((item) => ({ id: `incident-${item.id}`, icon: ShieldAlert, label: item.title || item.reference, detail: [item.reference, item.status].filter(Boolean).join(' · '), time: item.updated_at || item.created_at, severity: item.severity })),
    ...satelliteAnomalies.map((item) => ({ id: `rf-${item.id}`, icon: RadioTower, label: t('satelliteProfile.timelineRf', { metric: metricLabel(item.metric) }), detail: `${item.agent_id || '—'} · ${(Number(item.z_score) || 0).toFixed(1)}σ`, time: item.detected_at, severity: Math.abs(Number(item.z_score) || 0) >= 5 ? 'high' : 'medium' })),
    ...events.map((item) => ({ id: `orbital-${item.id}`, icon: Orbit, label: item.event_type, detail: [item.source, item.confidence != null ? `${Math.round(Number(item.confidence) * 100)}%` : ''].filter(Boolean).join(' · '), time: item.event_time, severity: item.severity })),
    ...exposureAlerts.map((item) => ({ id: `alert-${item.id}`, icon: TriangleAlert, label: item.cve_id, detail: item.matched_keyword || item.asset_value, time: item.created_at, severity: item.severity })),
  ].sort((a, b) => parseTime(b.time) - parseTime(a.time)).slice(0, 12)

  return (
    <section aria-labelledby="satellite-security-profile-heading" className="grid gap-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <button type="button" onClick={onBack} className="mb-3 inline-flex items-center gap-2 text-xs text-lyra-muted hover:text-lyra-goldBright"><ArrowLeft className="h-3.5 w-3.5" />{t('satelliteProfile.back')}</button>
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('satelliteProfile.eyebrow')}</p>
          <div className="mt-1 flex flex-wrap items-center gap-3">
            <h1 id="satellite-security-profile-heading" className="font-display text-3xl font-medium text-lyra-text">{satellite.name}</h1>
            <span className={`rounded-full border px-2.5 py-1 font-mono text-[10px] tracking-[0.14em] ${tone(satellite.tle_status)}`}>{satellite.tle_status || 'unknown'}</span>
          </div>
          <p className="mt-2 font-mono text-xs text-lyra-muted">NORAD {satellite.norad_id ?? '—'} · {t('satelliteProfile.updated')} {satellite.updated_at ? new Date(satellite.updated_at).toLocaleString() : '—'}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {access.incidents && <button type="button" onClick={() => onNavigate('incidents')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:text-lyra-goldBright">{t('satelliteProfile.openIncidents')}</button>}
          {access.rf && <button type="button" onClick={() => onNavigate('signal')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:text-lyra-goldBright">{t('satelliteProfile.openRf')}</button>}
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-[1.1fr_1fr]">
        <RiskGauge {...risk} t={t} />
        <div className="grid gap-3 sm:grid-cols-2">
          {[
            [Satellite, t('satelliteProfile.altitude'), satellite.position ? `${Number(satellite.position.altitude_km).toFixed(0)} km` : '—'],
            [ShieldAlert, t('satelliteProfile.openIncidentsCount'), access.incidents ? openIncidents.length : '—'],
            [RadioTower, t('satelliteProfile.rfAnomalies24h'), access.rf ? recentAnomalies.length : '—'],
            [Activity, t('satelliteProfile.relatedAgents'), access.agents ? relatedAgents.length : '—'],
          ].map(([Icon, label, value]) => (
            <div key={label} className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-4">
              <div className="flex items-center justify-between"><p className="text-[10px] tracking-[0.14em] text-lyra-mutedDim">{label}</p><Icon className="h-3.5 w-3.5 text-lyra-mutedDim" /></div>
              <p className="mt-2 font-mono text-2xl text-lyra-text">{value}</p>
            </div>
          ))}
        </div>
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-5">
          <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('satelliteProfile.orbitalEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg text-lyra-text">{t('satelliteProfile.orbitalTitle')}</h2>
          <dl className="mt-4 grid gap-3 text-sm">
            <div className="flex justify-between gap-4"><dt className="text-lyra-muted">{t('satelliteProfile.latitude')}</dt><dd className="font-mono text-lyra-text">{satellite.position ? `${Number(satellite.position.latitude_deg).toFixed(2)}°` : '—'}</dd></div>
            <div className="flex justify-between gap-4"><dt className="text-lyra-muted">{t('satelliteProfile.longitude')}</dt><dd className="font-mono text-lyra-text">{satellite.position ? `${Number(satellite.position.longitude_deg).toFixed(2)}°` : '—'}</dd></div>
            <div className="flex justify-between gap-4"><dt className="text-lyra-muted">{t('satelliteProfile.confidence')}</dt><dd className="font-mono text-lyra-text">{satellite.orbital_confidence || '—'}</dd></div>
            <div className="flex justify-between gap-4"><dt className="text-lyra-muted">{t('satelliteProfile.tleStatus')}</dt><dd className="font-mono text-lyra-text">{satellite.tle_status || '—'}</dd></div>
          </dl>
        </div>

        <div className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-5">
          <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('satelliteProfile.securityEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg text-lyra-text">{t('satelliteProfile.securityTitle')}</h2>
          <div className="mt-4 grid gap-3 text-sm">
            <div className="flex justify-between gap-4"><span className="text-lyra-muted">{t('satelliteProfile.incidents')}</span><span className="font-mono text-lyra-text">{access.incidents ? satelliteIncidents.length : '—'}</span></div>
            <div className="flex justify-between gap-4"><span className="text-lyra-muted">{t('satelliteProfile.rfAnomalies')}</span><span className="font-mono text-lyra-text">{access.rf ? satelliteAnomalies.length : '—'}</span></div>
            <div className="flex justify-between gap-4"><span className="text-lyra-muted">{t('satelliteProfile.orbitalEvents')}</span><span className="font-mono text-lyra-text">{orbitalEvents === null ? '…' : events.length}</span></div>
            <div className="flex justify-between gap-4"><span className="text-lyra-muted">{t('satelliteProfile.exposureMatches')}</span><span className="font-mono text-lyra-text">{access.alerts ? exposureAlerts.length : '—'}</span></div>
          </div>
          <p className="mt-4 text-[11px] leading-relaxed text-lyra-mutedDim">{t('satelliteProfile.exposureNote')}</p>
          {eventsError && <p className="mt-3 text-xs text-lyra-warn">{eventsError}</p>}
        </div>

        <div className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-5">
          <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('satelliteProfile.agentsEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg text-lyra-text">{t('satelliteProfile.agentsTitle')}</h2>
          <div className="mt-4 grid gap-2">
            {!access.agents ? <p className="text-xs text-lyra-mutedDim">{t('satelliteProfile.restricted')}</p> : relatedAgents.length ? relatedAgents.map((agent) => (
              <div key={agent.agent_id} className="rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2">
                <p className="font-mono text-xs text-lyra-text">{agent.agent_id}</p>
                <p className="mt-1 text-[10px] text-lyra-mutedDim">{agent.status || 'active'}{agent.last_seen_at ? ` · ${new Date(agent.last_seen_at).toLocaleString()}` : ''}</p>
              </div>
            )) : <p className="text-xs text-lyra-mutedDim">{t('satelliteProfile.noRelatedAgents')}</p>}
          </div>
        </div>
      </div>

      <SatelliteSources intelligence={sourceIntelligence} loading={!sourceIntelligence && !sourcesError} error={sourcesError} t={t} />

      <div className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-5">
        <div className="flex items-center justify-between gap-3">
          <div><p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('satelliteProfile.timelineEyebrow')}</p><h2 className="mt-1 font-display text-lg text-lyra-text">{t('satelliteProfile.timelineTitle')}</h2></div>
          <Clock3 className="h-4 w-4 text-lyra-mutedDim" />
        </div>
        {timeline.length ? <ul className="mt-3">{timeline.map((item) => <TimelineItem key={item.id} {...item} />)}</ul> : (
          <div className="mt-4 flex items-start gap-3 rounded border border-dashed border-lyra-border px-4 py-5"><ShieldCheck className="mt-0.5 h-4 w-4 text-lyra-good" /><div><p className="text-sm text-lyra-text">{t('satelliteProfile.timelineEmpty')}</p><p className="mt-1 text-xs text-lyra-mutedDim">{t('satelliteProfile.timelineEmptyHint')}</p></div></div>
        )}
      </div>
    </section>
  )
}
