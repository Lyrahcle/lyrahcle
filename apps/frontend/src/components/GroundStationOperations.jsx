import { useMemo, useState } from 'react'
import { Activity, ArrowLeft, Cpu, MapPin, RadioTower, Search, ShieldAlert, WifiOff } from 'lucide-react'
import { useI18n } from '../lib/I18nContext.jsx'
import { calculateGroundStationRisk, parseRiskTime } from '../lib/risk.js'
import { observedSatelliteIds, readingsForStation, stationCoordinates, visibleSatellites } from '../lib/groundStations.js'
import WorldMap from './WorldMap.jsx'
import SeverityBadge from './SeverityBadge.jsx'
import SignalBar from './SignalBar.jsx'
import TableScroll from './TableScroll.jsx'

const DAY_MS = 24 * 60 * 60 * 1000
const INACTIVE_INCIDENTS = new Set(['resolved', 'false_positive'])

function formatDate(value) {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString()
}

function statusClass(state) {
  if (state === 'online') return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
  if (state === 'attention') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-border bg-lyra-bgSoft text-lyra-muted'
}

function riskClass(level) {
  if (level === 'critical') return 'border-lyra-bad/30 bg-lyra-bad/10 text-lyra-bad'
  if (level === 'elevated') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
}

function stationEvidence(agent, incidents, anomalies, now) {
  const stationIncidents = incidents.filter((item) => item.agent_id === agent.agent_id)
  const stationAnomalies = anomalies.filter((item) => item.agent_id === agent.agent_id)
  return {
    incidents: stationIncidents,
    anomalies: stationAnomalies,
    activeIncidents: stationIncidents.filter((item) => !INACTIVE_INCIDENTS.has(item.status)),
    recentAnomalies: stationAnomalies.filter((item) => now - parseRiskTime(item.detected_at) <= DAY_MS),
  }
}

function StationDetail({ agent, satellites, readings, anomalies, incidents, access, onBack, onNavigate }) {
  const { t } = useI18n()
  const now = Date.now()
  const coordinates = stationCoordinates(agent)
  const evidence = stationEvidence(agent, incidents, anomalies, now)
  const stationReadings = readingsForStation(readings, agent.agent_id, now)
  const observedIds = observedSatelliteIds(readings, agent.agent_id, now)
  const observed = satellites.filter((item) => observedIds.includes(item.id))
  const visible = access.orbital ? visibleSatellites(agent, satellites) : []
  const risk = calculateGroundStationRisk({ agent, incidents: access.incidents ? evidence.incidents : [], anomalies: access.rf ? evidence.anomalies : [], now })
  const lastSeen = agent.last_seen_at || agent.device?.reported_at || agent.certificate?.last_seen_at

  return (
    <section className="grid gap-6" aria-labelledby="ground-station-detail-heading">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <button type="button" onClick={onBack} className="mb-3 inline-flex items-center gap-2 text-xs text-lyra-muted hover:text-lyra-goldBright">
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
            {t('groundStations.back')}
          </button>
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('groundStations.detailEyebrow')}</p>
          <h1 id="ground-station-detail-heading" className="mt-1 font-display text-2xl font-medium text-lyra-text">{agent.agent_id}</h1>
          <p className="mt-1 text-sm text-lyra-muted">{t('groundStations.detailDescription')}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {access.rf && <button type="button" onClick={() => onNavigate('signal')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:border-lyra-gold/40 hover:text-lyra-text">{t('groundStations.openRf')}</button>}
          {access.incidents && <button type="button" onClick={() => onNavigate('incidents')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:border-lyra-gold/40 hover:text-lyra-text">{t('groundStations.openIncidents')}</button>}
          <button type="button" onClick={() => onNavigate('connectivity')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:border-lyra-gold/40 hover:text-lyra-text">{t('groundStations.openFleet')}</button>
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4">
          <p className="text-[10px] tracking-[0.15em] text-lyra-muted">{t('groundStations.metric.status')}</p>
          <span className={`mt-2 inline-flex rounded-full border px-2 py-1 font-mono text-xs ${statusClass(risk.state)}`}>{t(`groundStations.state.${risk.state}`)}</span>
        </div>
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4">
          <p className="text-[10px] tracking-[0.15em] text-lyra-muted">{t('groundStations.metric.risk')}</p>
          <span className={`mt-2 inline-flex rounded-full border px-2 py-1 font-mono text-xs ${riskClass(risk.level)}`}>{risk.score} / 100</span>
        </div>
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4">
          <p className="text-[10px] tracking-[0.15em] text-lyra-muted">{t('groundStations.metric.visible')}</p>
          <p className="mt-2 font-mono text-2xl text-lyra-text">{access.orbital ? visible.length : '—'}</p>
        </div>
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4">
          <p className="text-[10px] tracking-[0.15em] text-lyra-muted">{t('groundStations.metric.observed')}</p>
          <p className="mt-2 font-mono text-2xl text-lyra-text">{access.rf ? observed.length : '—'}</p>
        </div>
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4">
          <p className="text-[10px] tracking-[0.15em] text-lyra-muted">{t('groundStations.metric.incidents')}</p>
          <p className="mt-2 font-mono text-2xl text-lyra-text">{access.incidents ? evidence.activeIncidents.length : '—'}</p>
        </div>
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4">
          <p className="text-[10px] tracking-[0.15em] text-lyra-muted">{t('groundStations.metric.anomalies')}</p>
          <p className="mt-2 font-mono text-2xl text-lyra-text">{access.rf ? evidence.recentAnomalies.length : '—'}</p>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/45 p-5">
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('groundStations.stationEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg font-medium text-lyra-text">{t('groundStations.stationTitle')}</h2>
          <dl className="mt-4 grid gap-4 sm:grid-cols-2">
            <div><dt className="text-xs text-lyra-muted">{t('groundStations.latitude')}</dt><dd className="mt-1 font-mono text-sm text-lyra-text">{coordinates ? `${coordinates.latitude.toFixed(5)}°` : '—'}</dd></div>
            <div><dt className="text-xs text-lyra-muted">{t('groundStations.longitude')}</dt><dd className="mt-1 font-mono text-sm text-lyra-text">{coordinates ? `${coordinates.longitude.toFixed(5)}°` : '—'}</dd></div>
            <div><dt className="text-xs text-lyra-muted">{t('groundStations.altitude')}</dt><dd className="mt-1 font-mono text-sm text-lyra-text">{coordinates ? `${coordinates.altitude.toFixed(0)} m` : '—'}</dd></div>
            <div><dt className="text-xs text-lyra-muted">{t('groundStations.lastContact')}</dt><dd className="mt-1 text-sm text-lyra-text">{formatDate(lastSeen)}</dd></div>
            <div><dt className="text-xs text-lyra-muted">{t('groundStations.hardware')}</dt><dd className="mt-1 text-sm text-lyra-text">{agent.device?.hardware || agent.device?.product || agent.device?.driver || '—'}</dd></div>
            <div><dt className="text-xs text-lyra-muted">{t('groundStations.integrity')}</dt><dd className="mt-1 text-sm text-lyra-text">{agent.device?.capabilities?.integrity_status || '—'}</dd></div>
          </dl>
          <div className="mt-5 border-t border-lyra-borderSoft pt-4">
            <p className="text-xs text-lyra-muted">{t('groundStations.riskFactors')}</p>
            {!risk.factors.length ? <p className="mt-2 text-sm text-lyra-good">{t('groundStations.noRiskFactors')}</p> : (
              <div className="mt-2 grid gap-2">
                {risk.factors.map((factor) => <div key={factor.key} className="flex items-center justify-between rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2 text-sm"><span className="text-lyra-text">{t(`groundStations.riskFactor.${factor.key}`)}</span><span className="font-mono text-lyra-warn">+{factor.points}</span></div>)}
              </div>
            )}
          </div>
        </div>

        <div className="rounded-lg border border-lyra-border bg-lyra-panel/45 p-5">
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('groundStations.visibilityEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg font-medium text-lyra-text">{t('groundStations.visibilityTitle')}</h2>
          {!access.orbital ? <p className="mt-4 text-sm text-lyra-muted">{t('groundStations.restricted')}</p> : !visible.length ? <p className="mt-4 text-sm text-lyra-muted">{t('groundStations.noVisible')}</p> : (
            <div className="mt-4 grid gap-2">
              {visible.slice(0, 8).map(({ satellite, elevation }) => (
                <div key={satellite.id} className="flex items-center justify-between rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2">
                  <div className="min-w-0"><p className="truncate text-sm text-lyra-text">{satellite.name}</p><p className="mt-0.5 font-mono text-[10px] text-lyra-mutedDim">NORAD {satellite.norad_id}</p></div>
                  <span className="font-mono text-sm text-lyra-goldBright">{elevation.toFixed(1)}°</span>
                </div>
              ))}
            </div>
          )}
          <p className="mt-4 text-xs text-lyra-mutedDim">{t('groundStations.visibilityMethod')}</p>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/45 p-5">
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('groundStations.observedEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg font-medium text-lyra-text">{t('groundStations.observedTitle')}</h2>
          {!access.rf ? <p className="mt-4 text-sm text-lyra-muted">{t('groundStations.restricted')}</p> : !observed.length ? <p className="mt-4 text-sm text-lyra-muted">{t('groundStations.noObserved')}</p> : (
            <div className="mt-4 flex flex-wrap gap-2">{observed.map((satellite) => <span key={satellite.id} className="rounded-full border border-lyra-border px-3 py-1 text-xs text-lyra-text">{satellite.name}</span>)}</div>
          )}
        </div>

        <div className="rounded-lg border border-lyra-border bg-lyra-panel/45 p-5">
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('groundStations.securityEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg font-medium text-lyra-text">{t('groundStations.securityTitle')}</h2>
          <div className="mt-4 grid gap-2">
            {access.incidents && evidence.activeIncidents.slice(0, 4).map((incident) => <div key={incident.id} className="flex items-start justify-between gap-3 rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2"><div className="min-w-0"><p className="truncate text-sm text-lyra-text">{incident.title}</p><p className="mt-1 text-[11px] text-lyra-mutedDim">{formatDate(incident.created_at)}</p></div><SeverityBadge severity={incident.severity} /></div>)}
            {access.rf && evidence.recentAnomalies.slice(0, 4).map((anomaly) => <div key={anomaly.id} className="flex items-center justify-between gap-3 rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2"><div className="min-w-0"><p className="truncate text-sm text-lyra-text">{t('groundStations.rfAnomaly')} · {anomaly.metric}</p><p className="mt-1 text-[11px] text-lyra-mutedDim">{formatDate(anomaly.detected_at)}</p></div><span className="font-mono text-xs text-lyra-warn">{Number(anomaly.z_score).toFixed(1)}σ</span></div>)}
            {((!access.incidents || !evidence.activeIncidents.length) && (!access.rf || !evidence.recentAnomalies.length)) && <p className="text-sm text-lyra-muted">{t('groundStations.noSecurityEvidence')}</p>}
          </div>
        </div>
      </div>

      {access.rf && (
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/45 p-5">
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('groundStations.rfEyebrow')}</p>
          <h2 className="mt-1 font-display text-lg font-medium text-lyra-text">{t('groundStations.rfTitle')}</h2>
          {!stationReadings.length ? <p className="mt-4 text-sm text-lyra-muted">{t('groundStations.noRf')}</p> : (
            <TableScroll>
              <table className="mt-4 w-full text-left text-sm">
                <thead><tr className="border-b border-lyra-borderSoft"><th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('groundStations.frequency')}</th><th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('groundStations.signal')}</th><th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('groundStations.satellite')}</th><th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('groundStations.captured')}</th></tr></thead>
                <tbody>{stationReadings.slice(0, 8).map((reading, index) => {
                  const satellite = satellites.find((item) => item.id === reading.satellite_id)
                  return <tr key={`${reading.captured_at}-${index}`} className="border-b border-lyra-borderSoft/60 last:border-0"><td className="py-3 font-mono text-xs text-lyra-text">{(Number(reading.frequency_hz) / 1e6).toFixed(3)} MHz</td><td className="py-3"><SignalBar dbm={reading.power_dbm} /></td><td className="py-3 text-xs text-lyra-text">{satellite?.name || '—'}</td><td className="py-3 text-xs text-lyra-mutedDim">{formatDate(reading.captured_at)}</td></tr>
                })}</tbody>
              </table>
            </TableScroll>
          )}
        </div>
      )}
    </section>
  )
}

export default function GroundStationOperations({ agents = [], satellites = [], readings = [], anomalies = [], incidents = [], access = {}, onNavigate = () => {} }) {
  const { t } = useI18n()
  const [selectedId, setSelectedId] = useState(null)
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState('all')
  const now = Date.now()
  const stations = useMemo(() => agents.filter((agent) => stationCoordinates(agent)), [agents])
  const enriched = stations.map((agent) => {
    const evidence = stationEvidence(agent, incidents, anomalies, now)
    const risk = calculateGroundStationRisk({ agent, incidents: access.incidents ? evidence.incidents : [], anomalies: access.rf ? evidence.anomalies : [], now })
    return { agent, risk, visible: access.orbital ? visibleSatellites(agent, satellites) : [], observed: access.rf ? observedSatelliteIds(readings, agent.agent_id, now) : [] }
  })
  const selected = enriched.find((item) => item.agent.agent_id === selectedId)?.agent ?? null

  if (selected) {
    return <StationDetail agent={selected} satellites={satellites} readings={readings} anomalies={anomalies} incidents={incidents} access={access} onBack={() => setSelectedId(null)} onNavigate={onNavigate} />
  }

  const counts = enriched.reduce((result, item) => {
    result[item.risk.state] += 1
    return result
  }, { online: 0, attention: 0, offline: 0 })
  const visibleSatelliteCount = new Set(enriched.flatMap((item) => item.visible.map(({ satellite }) => satellite.id))).size
  const filtered = enriched.filter((item) => {
    if (filter !== 'all' && item.risk.state !== filter) return false
    return item.agent.agent_id.toLowerCase().includes(query.trim().toLowerCase())
  })

  return (
    <section className="grid gap-6" aria-labelledby="ground-stations-heading">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('groundStations.eyebrow')}</p>
          <h1 id="ground-stations-heading" className="mt-1 font-display text-2xl font-medium text-lyra-text">{t('groundStations.title')}</h1>
          <p className="mt-1 max-w-3xl text-sm text-lyra-muted">{t('groundStations.description')}</p>
        </div>
        <button type="button" onClick={() => onNavigate('connectivity')} className="self-start rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:border-lyra-gold/40 hover:text-lyra-text">{t('groundStations.manageStations')}</button>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        {[
          [t('groundStations.metric.configured'), enriched.length, MapPin],
          [t('groundStations.metric.online'), counts.online, Activity],
          [t('groundStations.metric.attention'), counts.attention, ShieldAlert],
          [t('groundStations.metric.offline'), counts.offline, WifiOff],
          [t('groundStations.metric.visible'), access.orbital ? visibleSatelliteCount : '—', RadioTower],
        ].map(([label, value, Icon]) => <div key={label} className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4"><div className="flex items-center justify-between"><p className="text-[10px] tracking-[0.15em] text-lyra-muted">{label}</p><Icon className="h-3.5 w-3.5 text-lyra-mutedDim" aria-hidden="true" /></div><p className="mt-2 font-mono text-2xl text-lyra-text">{value}</p></div>)}
      </div>

      {!enriched.length ? (
        <div className="rounded-lg border border-dashed border-lyra-border px-6 py-12 text-center"><MapPin className="mx-auto h-6 w-6 text-lyra-mutedDim" aria-hidden="true" /><p className="mt-3 text-sm text-lyra-text">{t('groundStations.empty')}</p><p className="mt-1 text-xs text-lyra-muted">{t('groundStations.emptyHint')}</p></div>
      ) : (
        <>
          <div className="overflow-hidden rounded-lg border border-lyra-border bg-lyra-panel/40 p-3">
            <WorldMap satellites={access.orbital ? satellites : []} stations={stations} onStationSelect={(agent) => setSelectedId(agent.agent_id)} />
          </div>

          <div className="rounded-lg border border-lyra-border bg-lyra-panel/45 p-4">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div className="flex flex-wrap gap-2" role="group" aria-label={t('groundStations.filterLabel')}>
                {['all', 'online', 'attention', 'offline'].map((item) => <button key={item} type="button" aria-pressed={filter === item} onClick={() => setFilter(item)} className={`rounded border px-3 py-1.5 text-xs ${filter === item ? 'border-lyra-gold/40 bg-lyra-gold/10 text-lyra-goldBright' : 'border-lyra-border text-lyra-muted hover:text-lyra-text'}`}>{t(`groundStations.filter.${item}`)}</button>)}
              </div>
              <label className="relative block min-w-0 sm:min-w-64"><span className="sr-only">{t('groundStations.search')}</span><Search className="pointer-events-none absolute left-2.5 top-2.5 h-3.5 w-3.5 text-lyra-mutedDim" aria-hidden="true" /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t('groundStations.search')} className="w-full rounded border border-lyra-border bg-lyra-bgSoft py-2 pl-8 pr-3 text-sm text-lyra-text outline-none focus:border-lyra-gold/50" /></label>
            </div>

            {!filtered.length ? <p className="mt-4 text-sm text-lyra-muted">{t('groundStations.noMatches')}</p> : (
              <div className="mt-4 grid gap-3">
                {filtered.map(({ agent, risk, visible, observed }) => {
                  const coordinates = stationCoordinates(agent)
                  const lastSeen = agent.last_seen_at || agent.device?.reported_at || agent.certificate?.last_seen_at
                  return <button key={agent.agent_id} type="button" onClick={() => setSelectedId(agent.agent_id)} className="grid w-full gap-3 rounded-lg border border-lyra-borderSoft bg-lyra-bg/25 px-4 py-4 text-left transition hover:border-lyra-gold/35 md:grid-cols-[1.3fr_.7fr_.7fr_1fr_1fr] md:items-center"><div className="min-w-0"><p className="truncate font-mono text-sm text-lyra-goldBright">{agent.agent_id}</p><p className="mt-1 text-[11px] text-lyra-mutedDim">{coordinates.latitude.toFixed(4)}°, {coordinates.longitude.toFixed(4)}° · {coordinates.altitude.toFixed(0)} m</p></div><div><span className={`inline-flex rounded-full border px-2 py-0.5 font-mono text-[10px] ${statusClass(risk.state)}`}>{t(`groundStations.state.${risk.state}`)}</span></div><div><span className={`inline-flex rounded-full border px-2 py-0.5 font-mono text-[10px] ${riskClass(risk.level)}`}>{t('groundStations.riskValue', { score: risk.score })}</span></div><div><p className="text-xs text-lyra-text">{access.orbital ? t('groundStations.visibleValue', { count: visible.length }) : t('groundStations.restrictedShort')}</p><p className="mt-1 text-[11px] text-lyra-mutedDim">{access.rf ? t('groundStations.observedValue', { count: observed.length }) : t('groundStations.restrictedShort')}</p></div><div className="min-w-0"><p className="truncate text-xs text-lyra-text">{agent.device?.hardware || agent.device?.product || agent.device?.driver || t('groundStations.hardwareUnknown')}</p><p className="mt-1 text-[11px] text-lyra-mutedDim">{formatDate(lastSeen)}</p></div></button>
                })}
              </div>
            )}
          </div>
        </>
      )}
    </section>
  )
}
