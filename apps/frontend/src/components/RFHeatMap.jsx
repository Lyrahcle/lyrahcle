import { useMemo, useState } from 'react'
import DottedMap from 'dotted-map/without-countries'
import worldMapData from '../lib/world-map-data.json'
import WorldMap from './WorldMap.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

const POWER_FLOOR_DBM = -120
const POWER_CEILING_DBM = -30
const TIME_WINDOWS_MS = {
  all: null,
  '15m': 15 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
}

function finiteNumber(value) {
  if (value === '' || value === null || value === undefined) return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value))
}

function signalIntensity(powerDbm) {
  return clamp((powerDbm - POWER_FLOOR_DBM) / (POWER_CEILING_DBM - POWER_FLOOR_DBM), 0, 1)
}

function frequencyMHz(value) {
  return Number(value) / 1_000_000
}

function stationCoordinates(agent) {
  const latitude = finiteNumber(agent?.station?.latitude_deg)
  const longitude = finiteNumber(agent?.station?.longitude_deg)
  if (latitude === null || longitude === null || latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180) return null
  return { latitude, longitude }
}

export function buildRFHeatStations({ agents = [], readings = [], agentId = 'all', minMHz = '', maxMHz = '', timeWindow = 'all', now = Date.now() }) {
  const minFrequency = minMHz === '' ? null : finiteNumber(minMHz)
  const maxFrequency = maxMHz === '' ? null : finiteNumber(maxMHz)
  const invalidFrequencyRange = minFrequency !== null && maxFrequency !== null && minFrequency > maxFrequency
  const windowMs = TIME_WINDOWS_MS[timeWindow] ?? null
  const cutoff = windowMs === null ? null : now - windowMs

  const configured = agents
    .map((agent) => {
      const coordinates = stationCoordinates(agent)
      if (!coordinates || !agent?.agent_id) return null
      return { agentId: agent.agent_id, ...coordinates }
    })
    .filter(Boolean)
    .filter((station) => agentId === 'all' || station.agentId === agentId)

  const configuredIds = new Set(configured.map((station) => station.agentId))
  const grouped = new Map()

  if (!invalidFrequencyRange) {
    for (const reading of readings) {
      if (!configuredIds.has(reading?.agent_id)) continue
      const power = finiteNumber(reading?.power_dbm)
      const frequencyHz = finiteNumber(reading?.frequency_hz)
      const capturedAtMs = Date.parse(reading?.captured_at)
      if (power === null || frequencyHz === null || !Number.isFinite(capturedAtMs)) continue

      const mhz = frequencyMHz(frequencyHz)
      if (minFrequency !== null && mhz < minFrequency) continue
      if (maxFrequency !== null && mhz > maxFrequency) continue
      if (cutoff !== null && capturedAtMs < cutoff) continue

      const bucket = grouped.get(reading.agent_id) ?? []
      bucket.push({ power, frequencyHz, capturedAtMs })
      grouped.set(reading.agent_id, bucket)
    }
  }

  const stations = configured.map((station) => {
    const samples = grouped.get(station.agentId) ?? []
    if (!samples.length) return { ...station, sampleCount: 0, stats: null }

    let sum = 0
    let peak = -Infinity
    let minFrequencyHz = Infinity
    let maxFrequencyHz = -Infinity
    let latest = -Infinity
    for (const sample of samples) {
      sum += sample.power
      peak = Math.max(peak, sample.power)
      minFrequencyHz = Math.min(minFrequencyHz, sample.frequencyHz)
      maxFrequencyHz = Math.max(maxFrequencyHz, sample.frequencyHz)
      latest = Math.max(latest, sample.capturedAtMs)
    }
    const average = sum / samples.length
    return {
      ...station,
      sampleCount: samples.length,
      stats: {
        averagePowerDbm: average,
        peakPowerDbm: peak,
        minFrequencyHz,
        maxFrequencyHz,
        latestCapturedAtMs: latest,
        intensity: signalIntensity(average),
      },
    }
  })

  return {
    stations,
    activeCount: stations.filter((station) => station.stats).length,
    sampleCount: stations.reduce((total, station) => total + station.sampleCount, 0),
    invalidFrequencyRange,
  }
}

export default function RFHeatMap({ agents, readings }) {
  const { t } = useI18n()
  const [agentId, setAgentId] = useState('all')
  const [timeWindow, setTimeWindow] = useState('all')
  const [minMHz, setMinMHz] = useState('')
  const [maxMHz, setMaxMHz] = useState('')

  const map = useMemo(() => new DottedMap({ map: worldMapData }), [])
  const stationOptions = useMemo(
    () => (agents ?? [])
      .filter((agent) => agent?.agent_id && stationCoordinates(agent))
      .map((agent) => agent.agent_id)
      .sort((a, b) => a.localeCompare(b)),
    [agents],
  )

  const heat = useMemo(
    () => buildRFHeatStations({ agents, readings, agentId, minMHz, maxMHz, timeWindow }),
    [agents, readings, agentId, minMHz, maxMHz, timeWindow],
  )

  const markers = useMemo(
    () => heat.stations.map((station) => {
      const pin = map.getPin({ lat: station.latitude, lng: station.longitude })
      if (!pin) return null
      return {
        ...station,
        leftPct: (pin.x / map.width) * 100,
        topPct: (pin.y / map.height) * 100,
      }
    }).filter(Boolean),
    [heat.stations, map],
  )

  const resetFilters = () => {
    setAgentId('all')
    setTimeWindow('all')
    setMinMHz('')
    setMaxMHz('')
  }

  if (!stationOptions.length) {
    return <p className="text-sm text-lyra-mutedDim">{t('rfHeatMap.noStations')}</p>
  }

  return (
    <figure>
      <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-xs tracking-[0.16em] text-lyra-mutedDim">{t('rfHeatMap.title')}</p>
          <p className="mt-1 text-xs text-lyra-muted">{t('rfHeatMap.subtitle')}</p>
        </div>
        <span className="font-mono text-xs text-lyra-mutedDim">
          {t('rfHeatMap.summary', { active: heat.activeCount, configured: heat.stations.length, readings: heat.sampleCount })}
        </span>
      </div>

      <div className="mb-4 grid gap-3 rounded border border-lyra-borderSoft bg-lyra-bgSoft/40 p-3 md:grid-cols-2 xl:grid-cols-5">
        <label className="text-xs text-lyra-muted">
          <span className="mb-1 block">{t('rfHeatMap.agent')}</span>
          <select
            aria-label={t('rfHeatMap.agent')}
            value={agentId}
            onChange={(event) => setAgentId(event.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-panel px-2 py-1.5 font-mono text-xs text-lyra-text"
          >
            <option value="all">{t('rfHeatMap.allAgents')}</option>
            {stationOptions.map((id) => <option key={id} value={id}>{id}</option>)}
          </select>
        </label>

        <label className="text-xs text-lyra-muted">
          <span className="mb-1 block">{t('rfHeatMap.timeWindow')}</span>
          <select
            aria-label={t('rfHeatMap.timeWindow')}
            value={timeWindow}
            onChange={(event) => setTimeWindow(event.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-panel px-2 py-1.5 font-mono text-xs text-lyra-text"
          >
            <option value="all">{t('rfHeatMap.allTime')}</option>
            <option value="15m">{t('rfHeatMap.last15m')}</option>
            <option value="1h">{t('rfHeatMap.last1h')}</option>
            <option value="6h">{t('rfHeatMap.last6h')}</option>
            <option value="24h">{t('rfHeatMap.last24h')}</option>
          </select>
        </label>

        <label className="text-xs text-lyra-muted">
          <span className="mb-1 block">{t('rfHeatMap.minFrequency')}</span>
          <input
            aria-label={t('rfHeatMap.minFrequency')}
            type="number"
            min="0"
            step="0.001"
            inputMode="decimal"
            value={minMHz}
            onChange={(event) => setMinMHz(event.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-panel px-2 py-1.5 font-mono text-xs text-lyra-text"
          />
        </label>

        <label className="text-xs text-lyra-muted">
          <span className="mb-1 block">{t('rfHeatMap.maxFrequency')}</span>
          <input
            aria-label={t('rfHeatMap.maxFrequency')}
            type="number"
            min="0"
            step="0.001"
            inputMode="decimal"
            value={maxMHz}
            onChange={(event) => setMaxMHz(event.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-panel px-2 py-1.5 font-mono text-xs text-lyra-text"
          />
        </label>

        <div className="flex items-end">
          <button type="button" onClick={resetFilters} className="w-full rounded border border-lyra-border px-3 py-1.5 text-xs text-lyra-muted hover:border-lyra-gold hover:text-lyra-goldBright focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-lyra-gold">
            {t('rfHeatMap.reset')}
          </button>
        </div>
      </div>

      {heat.invalidFrequencyRange && (
        <p role="alert" className="mb-3 text-xs text-lyra-bad">{t('rfHeatMap.invalidRange')}</p>
      )}

      <div className="relative min-h-56 rounded border border-lyra-borderSoft bg-lyra-bgSoft p-2" role="region" aria-label={t('rfHeatMap.mapAria')}>
        <div aria-hidden="true"><WorldMap satellites={[]} /></div>
        {markers.map((marker) => {
          const active = Boolean(marker.stats)
          const intensity = marker.stats?.intensity ?? 0
          const haloSize = 28 + intensity * 56
          const haloOpacity = 0.12 + intensity * 0.5
          const ariaLabel = active
            ? t('rfHeatMap.markerAria', {
                agent: marker.agentId,
                average: marker.stats.averagePowerDbm.toFixed(1),
                peak: marker.stats.peakPowerDbm.toFixed(1),
                readings: marker.sampleCount,
              })
            : t('rfHeatMap.markerNoDataAria', { agent: marker.agentId })

          return (
            <div key={marker.agentId} className="group absolute z-10 -translate-x-1/2 -translate-y-1/2 hover:z-30 focus-within:z-30" style={{ left: `${marker.leftPct}%`, top: `${marker.topPct}%` }}>
              {active && (
                <span
                  aria-hidden="true"
                  className="pointer-events-none absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full bg-lyra-gold"
                  style={{ width: `${haloSize}px`, height: `${haloSize}px`, opacity: haloOpacity }}
                />
              )}
              <button
                type="button"
                aria-label={ariaLabel}
                className={`relative block h-3.5 w-3.5 rounded-full border outline-none focus-visible:ring-2 focus-visible:ring-lyra-gold ${active ? 'border-lyra-goldBright bg-lyra-gold' : 'border-lyra-mutedDim bg-lyra-panel'}`}
              />
              <div aria-hidden="true" className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 w-56 -translate-x-1/2 rounded border border-lyra-border bg-lyra-panel px-3 py-2 opacity-0 shadow-lg transition-opacity group-hover:opacity-100 group-focus-within:opacity-100">
                <p className="font-display text-xs font-medium text-lyra-goldBright">{marker.agentId}</p>
                <p className="mt-0.5 font-mono text-[10px] text-lyra-mutedDim">{marker.latitude.toFixed(2)}°, {marker.longitude.toFixed(2)}°</p>
                {active ? (
                  <>
                    <p className="mt-2 font-mono text-[10px] text-lyra-muted">{t('rfHeatMap.averagePower')}: {marker.stats.averagePowerDbm.toFixed(1)} dBm</p>
                    <p className="font-mono text-[10px] text-lyra-muted">{t('rfHeatMap.peakPower')}: {marker.stats.peakPowerDbm.toFixed(1)} dBm</p>
                    <p className="font-mono text-[10px] text-lyra-muted">{t('rfHeatMap.frequencyRange')}: {(marker.stats.minFrequencyHz / 1_000_000).toFixed(3)}–{(marker.stats.maxFrequencyHz / 1_000_000).toFixed(3)} MHz</p>
                    <p className="font-mono text-[10px] text-lyra-muted">{t('rfHeatMap.readings')}: {marker.sampleCount}</p>
                    <p className="font-mono text-[10px] text-lyra-muted">{t('rfHeatMap.lastCapture')}: {new Date(marker.stats.latestCapturedAtMs).toLocaleString()}</p>
                  </>
                ) : (
                  <p className="mt-2 text-[10px] text-lyra-mutedDim">{t('rfHeatMap.noData')}</p>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {heat.activeCount === 0 && !heat.invalidFrequencyRange && (
        <p className="mt-3 text-xs text-lyra-mutedDim">{t('rfHeatMap.noMatches')}</p>
      )}

      <div className="mt-3 flex flex-col gap-2 text-[10px] text-lyra-mutedDim sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2" aria-label={t('rfHeatMap.legendAria')}>
          <span>{t('rfHeatMap.legendWeak')}</span>
          <span aria-hidden="true" className="h-2 w-16 rounded-full bg-gradient-to-r from-lyra-gold/10 via-lyra-gold/40 to-lyra-gold" />
          <span>{t('rfHeatMap.legendStrong')}</span>
        </div>
        <p>{t('rfHeatMap.methodNote')}</p>
      </div>
    </figure>
  )
}
