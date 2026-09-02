import { useMemo } from 'react'
import DottedMap from 'dotted-map/without-countries'
import worldMapData from '../lib/world-map-data.json'
import { stationCoordinates } from '../lib/groundStations'

export default function WorldMap({ satellites = [], stations = [], onStationSelect = () => {} }) {
  const map = useMemo(() => new DottedMap({ map: worldMapData }), [])

  const baseMapImage = useMemo(() => {
    const svg = map.getSVG({
      radius: 0.24,
      color: '#2c2c30',
      shape: 'circle',
      backgroundColor: 'transparent',
    })
    return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
  }, [map])

  const markers = useMemo(
    () =>
      satellites
        .filter((s) => s.position)
        .map((s) => {
          const pin = map.getPin({
            lat: s.position.latitude_deg,
            lng: s.position.longitude_deg,
          })
          if (!pin) return null
          return {
            id: s.id,
            name: s.name,
            noradId: s.norad_id,
            position: s.position,
            leftPct: (pin.x / map.width) * 100,
            topPct: (pin.y / map.height) * 100,
          }
        })
        .filter(Boolean),
    [map, satellites]
  )

  const stationMarkers = useMemo(
    () => stations
      .map((agent) => {
        const station = stationCoordinates(agent)
        if (!station) return null
        const pin = map.getPin({ lat: station.latitude, lng: station.longitude })
        if (!pin) return null
        return {
          agent,
          station,
          leftPct: (pin.x / map.width) * 100,
          topPct: (pin.y / map.height) * 100,
        }
      })
      .filter(Boolean),
    [map, stations],
  )

  return (
    <div className="relative w-full">
      <img
        aria-hidden="true"
        alt=""
        src={baseMapImage}
        className="block h-auto w-full select-none"
        draggable="false"
      />


      {stationMarkers.map(({ agent, station, leftPct, topPct }) => (
        <div
          key={`station-${agent.agent_id}`}
          className="group absolute z-20 -translate-x-1/2 -translate-y-1/2"
          style={{ left: `${leftPct}%`, top: `${topPct}%` }}
        >
          <button
            type="button"
            onClick={() => onStationSelect(agent)}
            aria-label={`${agent.agent_id}, ground station, ${station.latitude.toFixed(4)}°, ${station.longitude.toFixed(4)}°`}
            className="relative flex h-5 w-5 items-center justify-center rounded-full border border-lyra-good/70 bg-lyra-panel shadow-[0_0_10px_rgba(87,191,146,0.35)] outline-none focus-visible:ring-2 focus-visible:ring-lyra-gold"
          >
            <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-lyra-good" />
          </button>
          <div aria-hidden="true" className="pointer-events-none absolute bottom-full left-1/2 z-30 mb-2 w-max max-w-56 -translate-x-1/2 rounded border border-lyra-border bg-lyra-panel px-3 py-2 opacity-0 shadow-lg transition-opacity duration-150 group-hover:opacity-100 group-focus-within:opacity-100">
            <p className="font-mono text-xs text-lyra-good">{agent.agent_id}</p>
            <p className="mt-1 font-mono text-[10px] text-lyra-muted">{station.latitude.toFixed(4)}°, {station.longitude.toFixed(4)}°</p>
            <p className="font-mono text-[10px] text-lyra-muted">{station.altitude.toFixed(0)} m</p>
          </div>
        </div>
      ))}

      {markers.map((m) => (
        <div
          key={m.id}
          className="group absolute z-0 -translate-x-1/2 -translate-y-1/2 hover:z-30 focus-within:z-30"
          style={{ left: `${m.leftPct}%`, top: `${m.topPct}%` }}
        >
          <span aria-hidden="true" className="pointer-events-none absolute inset-0 -m-1.5 motion-safe:animate-ping rounded-full bg-lyra-gold/40" />
          <button
            type="button"
            aria-label={`${m.name}, NORAD ${m.noradId}, ${m.position.latitude_deg.toFixed(2)}°, ${m.position.longitude_deg.toFixed(2)}°, ${m.position.altitude_km.toFixed(0)} km altitude`}
            className="relative block h-3.5 w-3.5 -m-0.5 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-lyra-gold"
          >
            <span aria-hidden="true" className="lyra-satellite-glow absolute inset-0 m-0.5 block rounded-full bg-lyra-goldBright shadow-[0_0_6px_rgba(240,217,168,0.8)]" />
          </button>

          <div aria-hidden="true" className="pointer-events-none absolute bottom-full left-1/2 z-10 mb-2 w-max -translate-x-1/2 rounded border border-lyra-border bg-lyra-panel px-3 py-2 opacity-0 shadow-lg transition-opacity duration-150 group-hover:opacity-100 group-focus-within:opacity-100">
            <p className="font-display text-xs font-medium text-lyra-goldBright">{m.name}</p>
            <p className="mt-0.5 font-mono text-[10px] text-lyra-mutedDim">
              NORAD {m.noradId}
            </p>
            <p className="mt-1 font-mono text-[10px] text-lyra-muted">
              {m.position.latitude_deg.toFixed(2)}°, {m.position.longitude_deg.toFixed(2)}°
            </p>
            <p className="font-mono text-[10px] text-lyra-muted">
              {m.position.altitude_km.toFixed(0)} km altitude
            </p>
          </div>
        </div>
      ))}
    </div>
  )
}
