export type TLEStatus = 'fresh' | 'aging' | 'stale' | 'unusable'
export type OrbitalConfidence = 'high' | 'degraded' | 'unavailable'
export type OrbitalEventType = 'maneuver' | 'conjunction'
export type SecuritySeverity = 'info' | 'low' | 'medium' | 'high' | 'critical'

export interface SatellitePosition {
  latitude_deg?: number
  longitude_deg?: number
  altitude_km?: number
}

export interface SatelliteRecord {
  id: string
  norad_id: number
  name: string
  tle_line1: string
  tle_line2: string
  tle_epoch?: string | null
  updated_at: string
}

export interface SatelliteWithPosition extends SatelliteRecord {
  tle_status?: TLEStatus
  tle_age_seconds?: number
  orbital_confidence?: OrbitalConfidence
  position?: SatellitePosition | null
}

export interface GroundStation {
  latitude_deg: number
  longitude_deg: number
  altitude_m: number
}

export interface SatellitePass {
  aos: string
  los: string
  peak_at: string
  max_elevation_deg: number
  duration_seconds: number
  aos_truncated?: boolean
  los_truncated?: boolean
}

export interface SatellitePassPredictionResponse {
  satellite_id: string
  norad_id: number
  satellite_name: string
  tle_status: 'fresh' | 'aging'
  tle_age_seconds: number
  orbital_confidence: 'high' | 'degraded'
  station: GroundStation
  from: string
  until: string
  min_elevation_deg: number
  passes: SatellitePass[]
}

export interface SatellitePassQuery {
  latitudeDeg: number
  longitudeDeg: number
  altitudeM?: number
  hours?: number
  minElevationDeg?: number
  limit?: number
}

export interface OrbitalEvent {
  id: string
  satellite_id: string
  event_type: OrbitalEventType
  event_time: string
  source: string
  confidence: number
  severity: SecuritySeverity
  metadata: Record<string, unknown>
  created_by?: string
  created_at: string
}

export interface CreateOrbitalEventInput {
  eventType: OrbitalEventType
  eventTime: string
  source: string
  confidence: number
  severity: SecuritySeverity
  metadata?: Record<string, unknown>
}

export interface SatelliteMutationInput {
  noradId: number
  name: string
  tleLine1: string
  tleLine2: string
}
