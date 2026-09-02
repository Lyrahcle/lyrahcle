import { useEffect, useMemo, useState } from 'react'
import type { ComponentType } from 'react'
import { Activity, Clock3, Orbit, RadioTower, Satellite, ShieldAlert, TriangleAlert } from 'lucide-react'
import WorldMap from './WorldMap.jsx'
import { fetchOrbitalEvents, fetchSatellitePasses } from '../lib/api'
import { calculateSatelliteRisk } from '../lib/risk'
import { satelliteElevationDeg, stationCoordinates } from '../lib/groundStations'
import { useI18n } from '../lib/I18nContext'
import CrossDomainAttackChain from './CrossDomainAttackChain'
import IncidentEvidenceCase from './IncidentEvidenceCase'

type Entity = Record<string, any>
type Page = { items?: Entity[] } | null

const MissionWorldMap = WorldMap as ComponentType<{
  satellites?: Entity[]
  stations?: Entity[]
  onStationSelect?: (station: Entity) => void
}>

const EARTH_RADIUS_KM = 6378.137
const INACTIVE_INCIDENT_STATUSES = new Set(['resolved', 'false_positive'])

function radians(value: number) { return (value * Math.PI) / 180 }
function finite(value: unknown): number | null {
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}
function rangeKm(station: ReturnType<typeof stationCoordinates>, position: any): number | null {
  if (!station || !position) return null
  const satLat = finite(position.latitude_deg)
  const satLon = finite(position.longitude_deg)
  const satAlt = finite(position.altitude_km)
  if (satLat === null || satLon === null || satAlt === null) return null
  const stationRadius = EARTH_RADIUS_KM + station.altitude / 1000
  const satelliteRadius = EARTH_RADIUS_KM + satAlt
  const cosine = Math.sin(radians(station.latitude)) * Math.sin(radians(satLat))
    + Math.cos(radians(station.latitude)) * Math.cos(radians(satLat)) * Math.cos(radians(satLon - station.longitude))
  return Math.sqrt(Math.max(0, stationRadius ** 2 + satelliteRadius ** 2 - 2 * stationRadius * satelliteRadius * cosine))
}
function timeValue(value: unknown) {
  const parsed = Date.parse(String(value || ''))
  return Number.isFinite(parsed) ? parsed : 0
}
function tone(level: string) {
  if (level === 'critical' || level === 'high') return 'border-lyra-bad/30 bg-lyra-bad/10 text-lyra-bad'
  if (level === 'elevated' || level === 'medium') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
}
function fmtDate(value: unknown) {
  const parsed = timeValue(value)
  return parsed ? new Date(parsed).toLocaleString() : '—'
}
function fmtHz(value: unknown) {
  const hz = finite(value)
  if (hz === null) return '—'
  if (Math.abs(hz) >= 1e9) return `${(hz / 1e9).toFixed(6)} GHz`
  if (Math.abs(hz) >= 1e6) return `${(hz / 1e6).toFixed(6)} MHz`
  if (Math.abs(hz) >= 1e3) return `${(hz / 1e3).toFixed(3)} kHz`
  return `${hz.toFixed(0)} Hz`
}

function Metric({ label, value, detail }: { label: string; value: string | number; detail?: string }) {
  return <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-4"><p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">{label}</p><p className="mt-2 font-mono text-xl text-lyra-text">{value}</p>{detail && <p className="mt-1 text-[11px] text-lyra-muted">{detail}</p>}</div>
}

export default function MissionControl({
  satellites = [], readings = [], anomalies = [], incidents = null, agents = [], assetAlerts = null,
  access = {}, onNavigate = () => {}, now = Date.now(), tmtcAnalysis = null, onIncidentCreated = () => {},
}: {
  satellites?: Entity[]; readings?: Entity[]; anomalies?: Entity[]; incidents?: Page; agents?: Entity[]; assetAlerts?: Page;
  access?: Record<string, boolean>; onNavigate?: (tab: any) => void; now?: number; tmtcAnalysis?: any; onIncidentCreated?: (incident: any) => void;
}) {
  const { t } = useI18n()
  const [selectedId, setSelectedId] = useState<string>('')
  const [passes, setPasses] = useState<Entity[] | null>(null)
  const [orbitalEvents, setOrbitalEvents] = useState<Entity[] | null>(null)

  useEffect(() => {
    if (!satellites.length) { setSelectedId(''); return }
    if (!selectedId || !satellites.some((satellite) => satellite.id === selectedId)) setSelectedId(satellites[0].id)
  }, [satellites, selectedId])

  const satellite = satellites.find((item) => item.id === selectedId) ?? null
  const satelliteReadings = useMemo(() => satellite && access.rf
    ? readings.filter((item) => item.satellite_id === satellite.id).sort((a, b) => timeValue(b.captured_at) - timeValue(a.captured_at)) : [], [access.rf, readings, satellite])
  const satelliteAnomalies = useMemo(() => satellite && access.rf
    ? anomalies.filter((item) => item.satellite_id === satellite.id).sort((a, b) => timeValue(b.detected_at) - timeValue(a.detected_at)) : [], [access.rf, anomalies, satellite])
  const satelliteIncidents = useMemo(() => satellite && access.incidents
    ? (incidents?.items ?? []).filter((item) => item.satellite_id === satellite.id).sort((a, b) => timeValue(b.updated_at || b.created_at) - timeValue(a.updated_at || a.created_at)) : [], [access.incidents, incidents, satellite])
  const exposureAlerts = useMemo(() => {
    if (!satellite || !access.alerts) return []
    const identities = new Set([String(satellite.name || '').toLowerCase(), String(satellite.norad_id ?? '').toLowerCase()])
    return (assetAlerts?.items ?? []).filter((item) => identities.has(String(item.asset_value || '').toLowerCase()))
  }, [access.alerts, assetAlerts, satellite])

  const relatedAgentId = satelliteReadings[0]?.agent_id || satelliteAnomalies[0]?.agent_id || satelliteIncidents.find((item) => item.agent_id)?.agent_id || ''
  const agent = access.agents ? agents.find((item) => item.agent_id === relatedAgentId) ?? null : null
  const coordinates = stationCoordinates(agent)
  const elevation = satellite ? satelliteElevationDeg(coordinates, satellite.position) : null
  const slantRange = satellite ? rangeKm(coordinates, satellite.position) : null

  useEffect(() => {
    let cancelled = false
    if (!satellite || !access.orbital) { setOrbitalEvents([]); return undefined }
    setOrbitalEvents(null)
    fetchOrbitalEvents(satellite.id, 168, 50)
      .then((data) => { if (!cancelled) setOrbitalEvents(Array.isArray(data) ? data : []) })
      .catch(() => { if (!cancelled) setOrbitalEvents([]) })
    return () => { cancelled = true }
  }, [satellite?.id, access.orbital])

  useEffect(() => {
    let cancelled = false
    if (!satellite || !coordinates || !access.orbital) { setPasses([]); return undefined }
    setPasses(null)
    fetchSatellitePasses(satellite.id, { latitudeDeg: coordinates.latitude, longitudeDeg: coordinates.longitude, altitudeM: coordinates.altitude, hours: 24, minElevationDeg: 0, limit: 3 })
      .then((data) => { if (!cancelled) setPasses(Array.isArray(data) ? data : []) })
      .catch(() => { if (!cancelled) setPasses([]) })
    return () => { cancelled = true }
  }, [satellite?.id, coordinates?.latitude, coordinates?.longitude, coordinates?.altitude, access.orbital])

  if (!satellites.length) return <section className="rounded-xl border border-lyra-border bg-lyra-panel/40 p-8 text-center"><Satellite className="mx-auto h-7 w-7 text-lyra-mutedDim" /><h1 className="mt-4 font-display text-2xl text-lyra-text">{t('missionControl.title')}</h1><p className="mt-2 text-sm text-lyra-muted">{t('missionControl.empty')}</p></section>
  if (!satellite) return null

  const risk = calculateSatelliteRisk({ satellite, incidents: satelliteIncidents, anomalies: satelliteAnomalies, orbitalEvents: orbitalEvents ?? [], exposureAlerts, now })
  const openIncidents = satelliteIncidents.filter((item) => !INACTIVE_INCIDENT_STATUSES.has(item.status)).length
  const latestRf = satelliteReadings[0] ?? null
  const nextPass = passes?.find((item) => timeValue(item.los_at || item.los || item.end_time) >= now) ?? passes?.[0] ?? null
  const aos = nextPass?.aos_at || nextPass?.aos || nextPass?.start_time
  const los = nextPass?.los_at || nextPass?.los || nextPass?.end_time
  const inContact = elevation !== null && elevation >= 0 && Boolean(latestRf && now - timeValue(latestRf.captured_at) <= 5 * 60 * 1000)
  const threat = risk.level === 'critical' ? 'critical' : risk.level === 'elevated' ? 'elevated' : 'nominal'
  const timeline = [
    ...satelliteIncidents.map((item) => ({ id: `i-${item.id}`, icon: ShieldAlert, label: item.title || item.reference || t('missionControl.incident'), detail: [item.reference, item.status].filter(Boolean).join(' · '), time: item.updated_at || item.created_at, severity: item.severity || 'medium' })),
    ...satelliteAnomalies.map((item) => ({ id: `a-${item.id}`, icon: RadioTower, label: t('missionControl.rfAnomaly'), detail: <><span>{item.agent_id || '—'}</span><span aria-hidden="true"> · </span><span>{(Number(item.z_score) || 0).toFixed(1)}σ</span></>, time: item.detected_at, severity: Math.abs(Number(item.z_score) || 0) >= 5 ? 'high' : 'medium' })),
    ...(orbitalEvents ?? []).map((item) => ({ id: `o-${item.id}`, icon: Orbit, label: item.event_type === 'conjunction' ? t('missionControl.conjunctionCandidate') : t('missionControl.maneuverCandidate'), detail: [item.source, item.confidence != null ? `${Math.round(Number(item.confidence) * 100)}%` : null].filter(Boolean).join(' · '), time: item.event_time, severity: item.severity || 'medium' })),
  ].sort((a, b) => timeValue(b.time) - timeValue(a.time)).slice(0, 8)

  return <section aria-labelledby="mission-control-heading" className="grid gap-6">
    <div className="rounded-xl border border-lyra-border bg-lyra-panel/45 p-5 sm:p-6">
      <div className="flex flex-col gap-5 xl:flex-row xl:items-start xl:justify-between">
        <div><p className="text-[10px] tracking-[0.18em] text-lyra-gold">{t('missionControl.eyebrow')}</p><div className="mt-2 flex flex-wrap items-center gap-3"><h1 id="mission-control-heading" className="font-display text-3xl font-medium text-lyra-text">{t('missionControl.title')}</h1><span className={`rounded-full border px-2.5 py-1 font-mono text-[10px] tracking-[0.12em] ${tone(inContact ? 'nominal' : 'medium')}`}>{inContact ? t('missionControl.inContact') : t('missionControl.notInContact')}</span></div><p className="mt-2 max-w-3xl text-sm text-lyra-muted">{t('missionControl.subtitle')}</p></div>
        <label className="min-w-64 text-xs text-lyra-muted"><span className="mb-1.5 block">{t('missionControl.satellite')}</span><select aria-label={t('missionControl.satellite')} value={selectedId} onChange={(event) => setSelectedId(event.target.value)} className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2.5 font-mono text-sm text-lyra-text outline-none focus:border-lyra-gold/50">{satellites.map((item) => <option key={item.id} value={item.id}>{item.name} · NORAD {item.norad_id ?? '—'}</option>)}</select></label>
      </div>
      <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        <Metric label={t('missionControl.missionRisk')} value={`${risk.score}/100`} detail={t(`missionControl.risk.${threat}`)} />
        <Metric label={t('missionControl.station')} value={access.agents ? (agent?.agent_id || '—') : t('missionControl.restricted')} />
        <Metric label={t('missionControl.elevation')} value={elevation === null ? '—' : `${elevation.toFixed(1)}°`} />
        <Metric label={t('missionControl.range')} value={slantRange === null ? '—' : `${slantRange.toFixed(0)} km`} />
        <Metric label="AOS" value={passes === null ? '…' : fmtDate(aos)} />
        <Metric label="LOS" value={passes === null ? '…' : fmtDate(los)} />
      </div>
    </div>

    <div className="grid gap-6 xl:grid-cols-[1.25fr_.75fr]">
      <div className="overflow-hidden rounded-xl border border-lyra-border bg-lyra-panel/40 p-3"><MissionWorldMap satellites={[satellite]} stations={agent ? [agent] : []} /></div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
        <Metric label={t('missionControl.openIncidents')} value={access.incidents ? openIncidents : '—'} />
        <Metric label={t('missionControl.rfAnomalies')} value={access.rf ? satelliteAnomalies.length : '—'} />
        <Metric label={t('missionControl.latestFrequency')} value={access.rf ? fmtHz(latestRf?.frequency_hz) : '—'} detail={latestRf ? fmtDate(latestRf.captured_at) : undefined} />
        <Metric label={t('missionControl.signalPower')} value={access.rf && finite(latestRf?.power_dbm) !== null ? `${Number(latestRf.power_dbm).toFixed(1)} dBm` : '—'} />
      </div>
    </div>

    <CrossDomainAttackChain anomalies={satelliteAnomalies} incidents={satelliteIncidents} orbitalEvents={orbitalEvents ?? []} tmtcAnalysis={tmtcAnalysis} />

    <IncidentEvidenceCase satellite={satellite} anomalies={satelliteAnomalies} incidents={satelliteIncidents} orbitalEvents={orbitalEvents ?? []} tmtcAnalysis={tmtcAnalysis} canManage={Boolean(access.incidentsManage)} agentId={agent?.agent_id || ''} onCreated={onIncidentCreated} />

    <div className="grid gap-6 lg:grid-cols-2">
      <div className="rounded-xl border border-lyra-border bg-lyra-panel/40 p-5"><div className="flex items-center justify-between gap-4"><div><p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('missionControl.securityEyebrow')}</p><h2 className="mt-1 font-display text-xl text-lyra-text">{t('missionControl.securityEvents')}</h2></div><ShieldAlert className="h-5 w-5 text-lyra-gold" /></div>{timeline.length ? <ul className="mt-4 divide-y divide-lyra-borderSoft">{timeline.map(({ id, icon: Icon, label, detail, time, severity }) => <li key={id} className="flex gap-3 py-3"><span className={`mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border ${tone(severity)}`}><Icon className="h-3.5 w-3.5" /></span><div className="min-w-0 flex-1"><p className="text-sm text-lyra-text">{label}</p><p className="mt-1 text-xs text-lyra-muted">{detail || '—'}</p><time className="mt-1 block font-mono text-[10px] text-lyra-mutedDim">{fmtDate(time)}</time></div></li>)}</ul> : <p className="mt-4 text-sm text-lyra-muted">{t('missionControl.noEvents')}</p>}</div>
      <div className="rounded-xl border border-lyra-border bg-lyra-panel/40 p-5"><div className="flex items-center justify-between gap-4"><div><p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">{t('missionControl.telemetryEyebrow')}</p><h2 className="mt-1 font-display text-xl text-lyra-text">{t('missionControl.liveContext')}</h2></div><Activity className="h-5 w-5 text-lyra-gold" /></div><dl className="mt-4 grid gap-3">{[[Orbit, 'NORAD', satellite.norad_id ?? '—'], [Clock3, t('missionControl.orbitUpdated'), fmtDate(satellite.updated_at)], [RadioTower, t('missionControl.lastCapture'), access.rf ? fmtDate(latestRf?.captured_at) : t('missionControl.restricted')], [TriangleAlert, t('missionControl.exposureAlerts'), access.alerts ? exposureAlerts.length : '—']].map(([Icon, label, value]: any[]) => <div key={String(label)} className="flex items-center justify-between gap-4 rounded border border-lyra-borderSoft bg-lyra-bg/25 px-3 py-3"><dt className="flex items-center gap-2 text-xs text-lyra-muted"><Icon className="h-3.5 w-3.5" />{label}</dt><dd className="font-mono text-xs text-lyra-text">{value}</dd></div>)}</dl><div className="mt-4 flex flex-wrap gap-2">{access.rf && <button type="button" onClick={() => onNavigate('signal')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:text-lyra-goldBright">{t('missionControl.openRf')}</button>}{access.incidents && <button type="button" onClick={() => onNavigate('incidents')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:text-lyra-goldBright">{t('missionControl.openIncidentWorkspace')}</button>}{access.agents && <button type="button" onClick={() => onNavigate('stations')} className="rounded border border-lyra-border px-3 py-2 text-xs text-lyra-muted hover:text-lyra-goldBright">{t('missionControl.openStations')}</button>}</div></div>
    </div>
  </section>
}
