const EARTH_RADIUS_KM = 6378.137
const DAY_MS = 24 * 60 * 60 * 1000

export interface StationCoordinates {
  latitude: number
  longitude: number
  altitude: number
}

export interface SatellitePosition {
  latitude_deg?: unknown
  longitude_deg?: unknown
  altitude_km?: unknown
}

export interface AgentStation {
  station?: {
    latitude_deg?: unknown
    longitude_deg?: unknown
    altitude_m?: unknown
  }
}

export interface Satellite {
  position?: SatellitePosition
  [key: string]: unknown
}

export interface VisibleSatelliteEntry {
  satellite: Satellite
  elevation: number
}

export interface StationReading {
  agent_id?: string
  captured_at?: string
  satellite_id?: string
  [key: string]: unknown
}

function radians(value: unknown): number {
  return (Number(value) * Math.PI) / 180
}

function finite(value: unknown): number | null {
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function vectorFromGeodetic(latitudeDeg: number, longitudeDeg: number, altitudeKm: number) {
  const latitude = radians(latitudeDeg)
  const longitude = radians(longitudeDeg)
  const radius = EARTH_RADIUS_KM + altitudeKm
  const cosLatitude = Math.cos(latitude)
  return {
    x: radius * cosLatitude * Math.cos(longitude),
    y: radius * cosLatitude * Math.sin(longitude),
    z: radius * Math.sin(latitude),
  }
}

export function stationCoordinates(agent: AgentStation | null | undefined): StationCoordinates | null {
  const latitude = finite(agent?.station?.latitude_deg)
  const longitude = finite(agent?.station?.longitude_deg)
  const altitude = finite(agent?.station?.altitude_m)
  if (latitude === null || longitude === null || altitude === null) return null
  if (latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180) return null
  return { latitude, longitude, altitude }
}

export function satelliteElevationDeg(
  station: StationCoordinates | null | undefined,
  position: SatellitePosition | null | undefined,
): number | null {
  if (!station || !position) return null
  const latitude = finite(position.latitude_deg)
  const longitude = finite(position.longitude_deg)
  const altitude = finite(position.altitude_km)
  if (latitude === null || longitude === null || altitude === null) return null
  const observer = vectorFromGeodetic(station.latitude, station.longitude, station.altitude / 1000)
  const satellite = vectorFromGeodetic(latitude, longitude, altitude)
  const dx = satellite.x - observer.x
  const dy = satellite.y - observer.y
  const dz = satellite.z - observer.z
  const range = Math.sqrt(dx * dx + dy * dy + dz * dz)
  const observerLength = Math.sqrt(observer.x * observer.x + observer.y * observer.y + observer.z * observer.z)
  if (!range || !observerLength) return null
  const dot = (dx * observer.x + dy * observer.y + dz * observer.z) / (range * observerLength)
  return (Math.asin(Math.max(-1, Math.min(1, dot))) * 180) / Math.PI
}

export function visibleSatellites(agent: AgentStation | null | undefined, satellites: Satellite[] = []): VisibleSatelliteEntry[] {
  const station = stationCoordinates(agent)
  if (!station) return []
  return satellites
    .map((satellite) => ({ satellite, elevation: satelliteElevationDeg(station, satellite.position) }))
    .filter((item): item is VisibleSatelliteEntry => item.elevation !== null && item.elevation >= 0)
    .sort((a, b) => b.elevation - a.elevation)
}

export function readingsForStation(readings: StationReading[] = [], agentId: string, now: number = Date.now()): StationReading[] {
  return readings
    .filter((item) => item.agent_id === agentId)
    .filter((item) => {
      const captured = Date.parse(item.captured_at ?? '')
      return Number.isFinite(captured) && now - captured <= DAY_MS
    })
    .sort((a, b) => Date.parse(b.captured_at ?? '') - Date.parse(a.captured_at ?? ''))
}

export function observedSatelliteIds(readings: StationReading[] = [], agentId: string, now: number = Date.now()): string[] {
  return [
    ...new Set(
      readingsForStation(readings, agentId, now)
        .map((item) => item.satellite_id)
        .filter((id): id is string => Boolean(id)),
    ),
  ]
}
