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

export type IncidentStatus = 'new' | 'investigating' | 'contained' | 'resolved' | 'false_positive'
export type IncidentEvidenceType = 'rf_anomaly' | 'request' | 'cve' | 'agent' | 'satellite' | 'external'

export interface IncidentRecord {
  id: string
  reference: string
  title: string
  description?: string
  severity: SecuritySeverity
  status: IncidentStatus
  source_type?: string
  updated_at: string
  agent_id?: string | null
  satellite_id?: string | null
  assigned_to?: string | null
}

export interface IncidentPage {
  items: IncidentRecord[]
  total: number
  limit: number
  offset: number
}

export interface IncidentEvent {
  id: string
  event_type: string
  created_at: string
  actor_username?: string | null
  message?: string | null
}

export interface IncidentEvidence {
  id: string
  evidence_type: IncidentEvidenceType | string
  evidence_ref: string
  label?: string | null
  created_at: string
  [key: string]: unknown
}

export interface IncidentDetail {
  incident: IncidentRecord | null
  events?: IncidentEvent[]
  evidence?: IncidentEvidence[]
  [key: string]: unknown
}

export interface UserSummary {
  id: string
  username?: string | null
  email: string
}

export interface RFReading {
  id?: string
  agent_id: string
  satellite_id?: string | null
  frequency_hz: number
  power_dbm: number
  captured_at: string
}

export interface RFAnomaly {
  id: string
  reading_id: string
  agent_id: string
  satellite_id?: string | null
  metric: string
  value: number
  baseline_mean: number
  baseline_stddev: number
  z_score: number
  detected_at: string
}

export interface RFDopplerComparison {
  anomaly_id: string
  reading_id: string
  agent_id: string
  satellite_id: string
  satellite_name: string
  observed_frequency_hz: number
  baseline_frequency_hz: number
  expected_doppler_hz: number
  expected_frequency_hz: number
  residual_hz: number
  tolerance_hz: number
  range_rate_mps: number
  elevation_deg: number
  visible: boolean
  tle_status: 'fresh' | 'aging'
  detected_at: string
}

