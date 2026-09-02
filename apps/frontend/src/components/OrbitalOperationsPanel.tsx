import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { createOrbitalEvent, fetchOrbitalEvents, fetchSatellitePasses } from '../lib/api'
import { hasPermission } from '../lib/permissions'
import type { OrbitalEvent, OrbitalEventType, SatellitePassPredictionResponse, SatelliteWithPosition, SecuritySeverity } from '../lib/apiTypes'
import { useI18n } from '../lib/I18nContext'
import SeverityBadge from './SeverityBadge.jsx'
import TableScroll from './TableScroll.jsx'

function formatDate(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString()
}

function formatDuration(seconds: number | string | null | undefined): string {
  const value = Number(seconds)
  if (!Number.isFinite(value)) return '—'
  const minutes = Math.floor(value / 60)
  const remainder = Math.round(value % 60)
  return `${minutes}m ${remainder}s`
}

interface OrbitalOperationsPanelProps {
  satellites?: SatelliteWithPosition[]
}

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback
}

export default function OrbitalOperationsPanel({ satellites = [] }: OrbitalOperationsPanelProps) {
  const { t } = useI18n()
  const canRead = hasPermission('orbital.read') || hasPermission('orbital.manage')
  const canManage = hasPermission('orbital.manage')
  const [satelliteId, setSatelliteId] = useState('')
  const [latitude, setLatitude] = useState('')
  const [longitude, setLongitude] = useState('')
  const [altitude, setAltitude] = useState('0')
  const [hours, setHours] = useState('24')
  const [minElevation, setMinElevation] = useState('10')
  const [prediction, setPrediction] = useState<SatellitePassPredictionResponse | null>(null)
  const [predicting, setPredicting] = useState(false)
  const [events, setEvents] = useState<OrbitalEvent[]>([])
  const [eventsLoading, setEventsLoading] = useState(false)
  const [error, setError] = useState('')

  const [eventType, setEventType] = useState<OrbitalEventType>('maneuver')
  const [eventTime, setEventTime] = useState('')
  const [eventSource, setEventSource] = useState('')
  const [eventConfidence, setEventConfidence] = useState('0.8')
  const [eventSeverity, setEventSeverity] = useState<SecuritySeverity>('info')
  const [creatingEvent, setCreatingEvent] = useState(false)

  const selectedSatellite = useMemo(
    () => satellites.find((satellite) => satellite.id === satelliteId) ?? null,
    [satellites, satelliteId],
  )

  useEffect(() => {
    if (!satellites.length) {
      setSatelliteId('')
      return
    }
    if (!satellites.some((satellite) => satellite.id === satelliteId)) {
      setSatelliteId(satellites[0].id)
    }
  }, [satellites, satelliteId])

  useEffect(() => {
    let cancelled = false
    setPrediction(null)
    if (!canRead || !satelliteId) {
      setEvents([])
      return undefined
    }
    setEventsLoading(true)
    fetchOrbitalEvents(satelliteId)
      .then((items) => {
        if (!cancelled) setEvents(Array.isArray(items) ? items : [])
      })
      .catch((err) => {
        if (!cancelled) setError(errorMessage(err, t('orbitalOperations.eventsFailed')))
      })
      .finally(() => {
        if (!cancelled) setEventsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [canRead, satelliteId, t])

  async function handlePredict(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setPredicting(true)
    try {
      const result = await fetchSatellitePasses(satelliteId, {
        latitudeDeg: Number(latitude),
        longitudeDeg: Number(longitude),
        altitudeM: Number(altitude || 0),
        hours: Number(hours),
        minElevationDeg: Number(minElevation),
        limit: 10,
      })
      setPrediction(result)
    } catch (err) {
      setPrediction(null)
      setError(errorMessage(err, t('orbitalOperations.predictionFailed')))
    } finally {
      setPredicting(false)
    }
  }

  async function reloadEvents() {
    const items = await fetchOrbitalEvents(satelliteId)
    setEvents(Array.isArray(items) ? items : [])
  }

  async function handleCreateEvent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setCreatingEvent(true)
    try {
      await createOrbitalEvent(satelliteId, {
        eventType,
        eventTime: new Date(eventTime).toISOString(),
        source: eventSource.trim(),
        confidence: Number(eventConfidence),
        severity: eventSeverity,
      })
      setEventSource('')
      await reloadEvents()
    } catch (err) {
      setError(errorMessage(err, t('orbitalOperations.eventCreateFailed')))
    } finally {
      setCreatingEvent(false)
    }
  }

  if (!canRead) {
    return <p className="text-sm text-lyra-muted">{t('orbitalOperations.noAccess')}</p>
  }

  if (!satellites.length) {
    return <p className="text-sm text-lyra-muted">{t('orbitalOperations.noSatellites')}</p>
  }

  return (
    <div className="grid gap-8">
      <div className="grid gap-6 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <form onSubmit={handlePredict} className="grid gap-4 rounded-lg border border-lyra-borderSoft bg-lyra-bg/30 p-4">
          <div>
            <h3 className="font-display text-base font-medium text-lyra-text">{t('orbitalOperations.passPrediction')}</h3>
            <p className="mt-1 text-xs text-lyra-muted">{t('orbitalOperations.passHint')}</p>
          </div>

          <label className="grid gap-1 text-xs text-lyra-muted">
            {t('orbitalOperations.satellite')}
            <select className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" value={satelliteId} onChange={(event) => setSatelliteId(event.target.value)}>
              {satellites.map((satellite) => <option key={satellite.id} value={satellite.id}>{satellite.name} · {satellite.norad_id}</option>)}
            </select>
          </label>

          <div className="grid gap-3 sm:grid-cols-3">
            <label className="grid gap-1 text-xs text-lyra-muted">
              {t('orbitalOperations.latitude')}
              <input required type="number" min="-90" max="90" step="any" value={latitude} onChange={(event) => setLatitude(event.target.value)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
            </label>
            <label className="grid gap-1 text-xs text-lyra-muted">
              {t('orbitalOperations.longitude')}
              <input required type="number" min="-180" max="180" step="any" value={longitude} onChange={(event) => setLongitude(event.target.value)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
            </label>
            <label className="grid gap-1 text-xs text-lyra-muted">
              {t('orbitalOperations.altitude')}
              <input type="number" min="-500" max="10000" step="any" value={altitude} onChange={(event) => setAltitude(event.target.value)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
            </label>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="grid gap-1 text-xs text-lyra-muted">
              {t('orbitalOperations.horizon')}
              <input required type="number" min="1" max="72" step="1" value={hours} onChange={(event) => setHours(event.target.value)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
            </label>
            <label className="grid gap-1 text-xs text-lyra-muted">
              {t('orbitalOperations.minElevation')}
              <input required type="number" min="0" max="90" step="0.1" value={minElevation} onChange={(event) => setMinElevation(event.target.value)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
            </label>
          </div>

          <button type="submit" disabled={predicting || !satelliteId} className="justify-self-start rounded border border-lyra-gold/40 bg-lyra-gold/10 px-4 py-2 text-sm text-lyra-goldBright disabled:opacity-50">
            {predicting ? t('orbitalOperations.predicting') : t('orbitalOperations.predict')}
          </button>
        </form>

        <div className="min-w-0 rounded-lg border border-lyra-borderSoft bg-lyra-bg/20 p-4">
          <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="font-display text-base font-medium text-lyra-text">{t('orbitalOperations.predictedPasses')}</h3>
              <p className="mt-1 text-xs text-lyra-muted">{selectedSatellite?.name ?? '—'}</p>
            </div>
            {prediction && (
              <div className="flex gap-2 font-mono text-[10px] tracking-wider">
                <span className="rounded border border-lyra-border px-2 py-1 text-lyra-muted">TLE {prediction.tle_status}</span>
                <span className="rounded border border-lyra-border px-2 py-1 text-lyra-muted">{prediction.orbital_confidence}</span>
              </div>
            )}
          </div>

          {!prediction ? (
            <p className="text-sm text-lyra-muted">{t('orbitalOperations.runPrediction')}</p>
          ) : !(prediction.passes ?? []).length ? (
            <p className="text-sm text-lyra-muted">{t('orbitalOperations.noPasses')}</p>
          ) : (
            <TableScroll>
              <table className="w-full text-left text-sm">
                <thead><tr className="border-b border-lyra-borderSoft">
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">AOS</th>
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">LOS</th>
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('orbitalOperations.peak')}</th>
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('orbitalOperations.duration')}</th>
                </tr></thead>
                <tbody>{prediction.passes.map((pass, index) => (
                  <tr key={`${pass.aos}-${index}`} className="border-b border-lyra-borderSoft/60 last:border-0">
                    <td className="py-3 pr-3 text-xs text-lyra-text">{formatDate(pass.aos)}</td>
                    <td className="py-3 pr-3 text-xs text-lyra-text">{formatDate(pass.los)}</td>
                    <td className="py-3 pr-3 font-mono text-xs text-lyra-goldBright">{Number(pass.max_elevation_deg).toFixed(1)}°</td>
                    <td className="py-3 font-mono text-xs text-lyra-muted">{formatDuration(pass.duration_seconds)}</td>
                  </tr>
                ))}</tbody>
              </table>
            </TableScroll>
          )}
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(300px,0.8fr)]">
        <div className="min-w-0 rounded-lg border border-lyra-borderSoft bg-lyra-bg/20 p-4">
          <div className="mb-4">
            <h3 className="font-display text-base font-medium text-lyra-text">{t('orbitalOperations.events')}</h3>
            <p className="mt-1 text-xs text-lyra-muted">{t('orbitalOperations.eventsHint')}</p>
          </div>
          {eventsLoading ? (
            <p className="text-sm text-lyra-muted">{t('common.loading')}</p>
          ) : !events.length ? (
            <p className="text-sm text-lyra-muted">{t('orbitalOperations.noEvents')}</p>
          ) : (
            <TableScroll>
              <table className="w-full text-left text-sm">
                <thead><tr className="border-b border-lyra-borderSoft">
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('orbitalOperations.type')}</th>
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('orbitalOperations.time')}</th>
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('orbitalOperations.source')}</th>
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('orbitalOperations.confidence')}</th>
                  <th className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('orbitalOperations.severity')}</th>
                </tr></thead>
                <tbody>{events.map((item) => (
                  <tr key={item.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                    <td className="py-3 pr-3 capitalize text-lyra-goldBright">{item.event_type}</td>
                    <td className="py-3 pr-3 text-xs text-lyra-text">{formatDate(item.event_time)}</td>
                    <td className="py-3 pr-3 text-xs text-lyra-muted">{item.source}</td>
                    <td className="py-3 pr-3 font-mono text-xs text-lyra-text">{Math.round(Number(item.confidence) * 100)}%</td>
                    <td className="py-3"><SeverityBadge severity={item.severity} /></td>
                  </tr>
                ))}</tbody>
              </table>
            </TableScroll>
          )}
        </div>

        {canManage && (
          <form onSubmit={handleCreateEvent} className="grid content-start gap-3 rounded-lg border border-lyra-borderSoft bg-lyra-bg/30 p-4">
            <div>
              <h3 className="font-display text-base font-medium text-lyra-text">{t('orbitalOperations.addEvent')}</h3>
              <p className="mt-1 text-xs text-lyra-muted">{t('orbitalOperations.candidateOnly')}</p>
            </div>
            <label className="grid gap-1 text-xs text-lyra-muted">{t('orbitalOperations.type')}
              <select value={eventType} onChange={(event) => setEventType(event.target.value as OrbitalEventType)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text">
                <option value="maneuver">{t('orbitalOperations.maneuver')}</option>
                <option value="conjunction">{t('orbitalOperations.conjunction')}</option>
              </select>
            </label>
            <label className="grid gap-1 text-xs text-lyra-muted">{t('orbitalOperations.time')}
              <input required type="datetime-local" value={eventTime} onChange={(event) => setEventTime(event.target.value)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
            </label>
            <label className="grid gap-1 text-xs text-lyra-muted">{t('orbitalOperations.source')}
              <input required maxLength={160} value={eventSource} onChange={(event) => setEventSource(event.target.value)} placeholder={t('orbitalOperations.sourcePlaceholder')} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
            </label>
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
              <label className="grid gap-1 text-xs text-lyra-muted">{t('orbitalOperations.confidence')}
                <input required type="number" min="0" max="1" step="0.01" value={eventConfidence} onChange={(event) => setEventConfidence(event.target.value)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text" />
              </label>
              <label className="grid gap-1 text-xs text-lyra-muted">{t('orbitalOperations.severity')}
                <select value={eventSeverity} onChange={(event) => setEventSeverity(event.target.value as SecuritySeverity)} className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text">
                  {['info', 'low', 'medium', 'high', 'critical'].map((severity) => <option key={severity} value={severity}>{severity}</option>)}
                </select>
              </label>
            </div>
            <button type="submit" disabled={creatingEvent || !eventTime || !eventSource.trim()} className="justify-self-start rounded border border-lyra-gold/40 bg-lyra-gold/10 px-4 py-2 text-sm text-lyra-goldBright disabled:opacity-50">
              {creatingEvent ? t('orbitalOperations.creating') : t('orbitalOperations.createEvent')}
            </button>
          </form>
        )}
      </div>

      {error && <p role="alert" className="rounded border border-lyra-bad/30 bg-lyra-bad/10 px-3 py-2 text-sm text-lyra-bad">{error}</p>}
    </div>
  )
}
