const DAY_MS = 24 * 60 * 60 * 1000
const AGENT_ONLINE_WINDOW_MS = 5 * 60 * 1000
const INACTIVE_INCIDENT_STATUSES = new Set(['resolved', 'false_positive'])

export type RiskLevelValue = 'critical' | 'elevated' | 'nominal'
export type RiskDomain = 'orbital' | 'security' | 'rf' | 'exposure' | 'platform' | 'ground'
export type AgentState = 'attention' | 'offline' | 'online'

export interface RiskFactor {
  key: string
  points: number
  domain: RiskDomain
}

export interface RiskResult {
  score: number
  level: RiskLevelValue
  factors: RiskFactor[]
}

export interface MissionRiskResult extends RiskResult {
  domains: Partial<Record<RiskDomain, number>>
}

export interface GroundStationRiskResult extends RiskResult {
  state: AgentState
}

interface RiskItem {
  status?: string
  severity?: string
  detected_at?: string
  z_score?: unknown
  acknowledged?: boolean
  [key: string]: unknown
}

interface SatelliteInput {
  tle_status?: string
  [key: string]: unknown
}

interface AgentInput {
  status?: string
  device?: {
    capabilities?: { integrity_status?: string }
    compatibility?: string
    reported_at?: string
  }
  last_seen_at?: string
  certificate?: { last_seen_at?: string }
  [key: string]: unknown
}

export function parseRiskTime(value: string | undefined): number {
  const parsed = Date.parse(value ?? '')
  return Number.isFinite(parsed) ? parsed : 0
}

export function riskSeverityRank(value: unknown): number {
  const ranks: Record<string, number> = { critical: 4, high: 3, medium: 2, low: 1, info: 0 }
  return ranks[String(value || '').toLowerCase()] ?? 0
}

export function riskLevel(score: number): RiskLevelValue {
  if (score >= 70) return 'critical'
  if (score >= 35) return 'elevated'
  return 'nominal'
}

export function calculateSatelliteRisk({
  satellite,
  incidents = [],
  anomalies = [],
  orbitalEvents = [],
  exposureAlerts = [],
  now = Date.now(),
}: {
  satellite?: SatelliteInput
  incidents?: RiskItem[]
  anomalies?: RiskItem[]
  orbitalEvents?: RiskItem[]
  exposureAlerts?: RiskItem[]
  now?: number
}): RiskResult {
  const factors: RiskFactor[] = []
  const tleStatus = String(satellite?.tle_status || '').toLowerCase()
  if (tleStatus === 'unusable') factors.push({ key: 'tleUnusable', points: 30, domain: 'orbital' })
  else if (tleStatus === 'stale') factors.push({ key: 'tleStale', points: 20, domain: 'orbital' })
  else if (tleStatus === 'aging') factors.push({ key: 'tleAging', points: 10, domain: 'orbital' })

  const activeIncidents = incidents.filter((item) => !INACTIVE_INCIDENT_STATUSES.has(item.status ?? ''))
  const criticalCount = activeIncidents.filter((item) => item.severity === 'critical').length
  const highCount = activeIncidents.filter((item) => item.severity === 'high').length
  if (criticalCount) factors.push({ key: 'criticalIncident', points: Math.min(30, criticalCount * 20), domain: 'security' })
  if (highCount) factors.push({ key: 'highIncident', points: Math.min(20, highCount * 10), domain: 'security' })

  const recentAnomalies = anomalies.filter((item) => now - parseRiskTime(item.detected_at) <= DAY_MS)
  const severeAnomalies = recentAnomalies.filter((item) => Math.abs(Number(item.z_score) || 0) >= 5).length
  if (severeAnomalies) factors.push({ key: 'rfAnomaly', points: Math.min(20, severeAnomalies * 8), domain: 'rf' })

  const highOrbital = orbitalEvents.filter((item) => riskSeverityRank(item.severity) >= riskSeverityRank('high')).length
  if (highOrbital) factors.push({ key: 'orbitalEvent', points: Math.min(15, highOrbital * 5), domain: 'orbital' })

  const urgentExposure = exposureAlerts.filter(
    (item) => !item.acknowledged && riskSeverityRank(item.severity) >= riskSeverityRank('high'),
  ).length
  if (urgentExposure) factors.push({ key: 'exposure', points: Math.min(15, urgentExposure * 5), domain: 'exposure' })

  const score = Math.min(100, factors.reduce((sum, item) => sum + item.points, 0))
  return { score, level: riskLevel(score), factors }
}

export function agentState(agent: AgentInput | null | undefined, now: number): AgentState {
  if (agent?.status === 'quarantined' || agent?.status === 'revoked') return 'attention'
  const integrity = agent?.device?.capabilities?.integrity_status
  const compatibility = agent?.device?.compatibility
  if (integrity === 'changed' || compatibility === 'unsupported' || compatibility === 'degraded') return 'attention'
  const lastSeen = parseRiskTime(agent?.last_seen_at || agent?.device?.reported_at || agent?.certificate?.last_seen_at)
  if (!lastSeen || now - lastSeen > AGENT_ONLINE_WINDOW_MS) return 'offline'
  return 'online'
}

export function calculateMissionRisk({
  satellites = [],
  incidents = [],
  anomalies = [],
  agents = [],
  alerts = [],
  backendStatus = 'checking',
  now = Date.now(),
}: {
  satellites?: SatelliteInput[]
  incidents?: RiskItem[]
  anomalies?: RiskItem[]
  agents?: AgentInput[]
  alerts?: RiskItem[]
  backendStatus?: string
  now?: number
}): MissionRiskResult {
  const factors: RiskFactor[] = []
  const activeIncidents = incidents.filter((item) => !INACTIVE_INCIDENT_STATUSES.has(item.status ?? ''))
  const criticalIncidents = activeIncidents.filter((item) => item.severity === 'critical').length
  const highIncidents = activeIncidents.filter((item) => item.severity === 'high').length
  const recentAnomalies = anomalies.filter((item) => now - parseRiskTime(item.detected_at) <= DAY_MS)
  const severeAnomalies = recentAnomalies.filter((item) => Math.abs(Number(item.z_score) || 0) >= 5).length
  const attentionAgents = agents.filter((agent) => agentState(agent, now) !== 'online').length
  const urgentAlerts = alerts.filter(
    (item) => !item.acknowledged && riskSeverityRank(item.severity) >= riskSeverityRank('high'),
  ).length
  const degradedOrbit = satellites.filter((item) =>
    ['aging', 'stale', 'unusable'].includes(String(item.tle_status || '').toLowerCase()),
  ).length

  if (backendStatus === 'offline') factors.push({ key: 'backendOffline', points: 30, domain: 'platform' })
  if (criticalIncidents) factors.push({ key: 'criticalIncidents', points: Math.min(35, criticalIncidents * 20), domain: 'security' })
  if (highIncidents) factors.push({ key: 'highIncidents', points: Math.min(20, highIncidents * 8), domain: 'security' })
  if (severeAnomalies) factors.push({ key: 'severeRf', points: Math.min(20, severeAnomalies * 5), domain: 'rf' })
  if (urgentAlerts) factors.push({ key: 'urgentExposure', points: Math.min(15, urgentAlerts * 5), domain: 'exposure' })
  if (attentionAgents) factors.push({ key: 'agentAttention', points: Math.min(15, attentionAgents * 4), domain: 'ground' })
  if (degradedOrbit) factors.push({ key: 'orbitalQuality', points: Math.min(15, degradedOrbit * 4), domain: 'orbital' })

  const score = Math.min(100, factors.reduce((sum, item) => sum + item.points, 0))
  const domains = factors.reduce<Partial<Record<RiskDomain, number>>>((result, factor) => {
    result[factor.domain] = Math.min(100, (result[factor.domain] || 0) + factor.points)
    return result
  }, {})
  return { score, level: riskLevel(score), factors, domains }
}

export function calculateGroundStationRisk({
  agent,
  incidents = [],
  anomalies = [],
  now = Date.now(),
}: {
  agent?: AgentInput
  incidents?: RiskItem[]
  anomalies?: RiskItem[]
  now?: number
}): GroundStationRiskResult {
  const factors: RiskFactor[] = []
  const state = agentState(agent, now)
  if (state === 'attention') factors.push({ key: 'agentAttention', points: 30, domain: 'ground' })
  else if (state === 'offline') factors.push({ key: 'agentOffline', points: 20, domain: 'ground' })

  const activeIncidents = incidents.filter((item) => !INACTIVE_INCIDENT_STATUSES.has(item.status ?? ''))
  const criticalCount = activeIncidents.filter((item) => item.severity === 'critical').length
  const highCount = activeIncidents.filter((item) => item.severity === 'high').length
  if (criticalCount) factors.push({ key: 'criticalIncident', points: Math.min(30, criticalCount * 20), domain: 'security' })
  if (highCount) factors.push({ key: 'highIncident', points: Math.min(20, highCount * 10), domain: 'security' })

  const recentAnomalies = anomalies.filter((item) => now - parseRiskTime(item.detected_at) <= DAY_MS)
  const severeAnomalies = recentAnomalies.filter((item) => Math.abs(Number(item.z_score) || 0) >= 5).length
  if (severeAnomalies) factors.push({ key: 'rfAnomaly', points: Math.min(20, severeAnomalies * 8), domain: 'rf' })

  const score = Math.min(100, factors.reduce((sum, item) => sum + item.points, 0))
  return { score, level: riskLevel(score), factors, state }
}
