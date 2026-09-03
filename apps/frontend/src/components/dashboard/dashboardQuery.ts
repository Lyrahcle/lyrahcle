import {
  fetchAgentKeys,
  fetchAgentIdentities,
  fetchIncidents,
  fetchAssets,
  fetchPrograms,
  fetchAssetAlerts,
  fetchAuditLog,
  fetchPing,
  fetchRfAnomalies,
  fetchRfDoppler,
  fetchRfReadings,
  fetchDetectionRules,
  fetchSatellites,
  fetchSpaceCVEs,
  fetchUsers,
  fetchOrganizations,
  fetchPlatformUsers,
  fetchRBACPermissions,
  fetchRBACRoles,
  UnauthorizedError,
} from '../../lib/api'
import type { IncidentFilters, SpaceCVEFilters } from '../../lib/api'
import type { RouteTab } from '../../lib/routeTabs'

const PAGE_LIMIT = 50

export type DashboardQueryInput = {
  activeTab: RouteTab
  assetsOffset: number
  cvesOffset: number
  cveFilters: SpaceCVEFilters
  alertsOffset: number
  incidentsOffset: number
  incidentFilters: IncidentFilters
  auditOffset: number
  access: {
    admin: boolean
    adminAccess: boolean
    audit: boolean
    assets: boolean
    alerts: boolean
    connectivity: boolean
    incidents: boolean
    incidentsManage: boolean
    orbital: boolean
    platform: boolean
    programs: boolean
    rbac: boolean
    rf: boolean
    users: boolean
  }
}

export type DashboardQueryResult = Record<string, any> & { backendStatus: 'online' | 'offline'; dbStatus?: 'online' | 'offline' }

async function optional<T>(fn: () => Promise<T>, fallback: T): Promise<T> {
  try {
    const value = await fn()
    return value ?? fallback
  } catch (error) {
    if (error instanceof UnauthorizedError) throw error
    return fallback
  }
}

export async function fetchDashboardTab(input: DashboardQueryInput): Promise<DashboardQueryResult> {
  const { activeTab, access } = input
  let backendStatus: 'online' | 'offline' = 'online'
  try { await fetchPing() } catch { backendStatus = 'offline' }
  const result: DashboardQueryResult = { backendStatus }

  if (activeTab === 'mission') {
    result.satellites = await optional(fetchSatellites, [])
    if (access.rf) [result.rfReadings, result.rfAnomalies] = await Promise.all([optional(fetchRfReadings, []), optional(fetchRfAnomalies, [])])
    if (access.incidents) result.incidents = await optional(() => fetchIncidents(PAGE_LIMIT, 0, {}), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
    if (access.connectivity) result.agentIdentities = await optional(fetchAgentIdentities, [])
    if (access.alerts) result.assetAlerts = await optional(() => fetchAssetAlerts(PAGE_LIMIT, 0), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
    return result
  }

  if (activeTab === 'orbital') {
    result.satellites = await optional(fetchSatellites, [])
    if (access.incidents) result.incidents = await optional(() => fetchIncidents(PAGE_LIMIT, 0, {}), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
    if (access.rf) result.rfAnomalies = await optional(fetchRfAnomalies, [])
    if (access.connectivity) result.agentIdentities = await optional(fetchAgentIdentities, [])
    if (access.alerts) result.assetAlerts = await optional(() => fetchAssetAlerts(PAGE_LIMIT, 0), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
    return result
  }

  if (activeTab === 'graph') {
    const tasks: Promise<void>[] = []
    const assign = <T,>(key: string, promise: Promise<T>) => tasks.push(promise.then((value) => { result[key] = value }))
    if (access.orbital) assign('satellites', optional(fetchSatellites, []))
    if (access.connectivity) assign('agentIdentities', optional(fetchAgentIdentities, []))
    if (access.rf) { assign('rfReadings', optional(fetchRfReadings, [])); assign('rfAnomalies', optional(fetchRfAnomalies, [])) }
    if (access.incidents) assign('incidents', optional(() => fetchIncidents(PAGE_LIMIT, 0, {}), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
    if (access.alerts) assign('assetAlerts', optional(() => fetchAssetAlerts(PAGE_LIMIT, 0), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
    if (access.assets) assign('assets', optional(() => fetchAssets(PAGE_LIMIT, 0), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
    if (access.programs) assign('programs', optional(fetchPrograms, []))
    assign('spaceCVEs', optional(() => fetchSpaceCVEs(PAGE_LIMIT, 0, { severity: '', keyword: '', publishedFrom: '', publishedTo: '', sort: 'published', order: 'desc' }), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
    await Promise.all(tasks)
    return result
  }

  if (activeTab === 'assets') {
    try {
      [result.assets, result.programs] = await Promise.all([
        fetchAssets(PAGE_LIMIT, input.assetsOffset),
        access.programs ? fetchPrograms() : Promise.resolve([]),
      ])
      result.dbStatus = 'online'
    } catch (error) {
      if (error instanceof UnauthorizedError) throw error
      result.assets = { items: [], total: 0, limit: PAGE_LIMIT, offset: input.assetsOffset }
      result.programs = []
      result.dbStatus = 'offline'
    }
    return result
  }

  if (activeTab === 'stations' && access.connectivity) {
    result.agentIdentities = await optional(fetchAgentIdentities, [])
    if (access.orbital) result.satellites = await optional(fetchSatellites, [])
    if (access.rf) [result.rfReadings, result.rfAnomalies] = await Promise.all([optional(fetchRfReadings, []), optional(fetchRfAnomalies, [])])
    if (access.incidents) result.incidents = await optional(() => fetchIncidents(PAGE_LIMIT, 0, {}), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
    return result
  }

  if (activeTab === 'signal') {
    [result.rfReadings, result.rfAnomalies, result.rfDoppler, result.agentIdentities, result.detectionRules] = await Promise.all([
      optional(fetchRfReadings, []), optional(fetchRfAnomalies, []), optional(fetchRfDoppler, []), optional(fetchAgentIdentities, []),
      optional(() => fetchDetectionRules(PAGE_LIMIT, 0), { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }),
    ])
    return result
  }

  if (activeTab === 'incidents') {
    result.incidents = await optional(() => fetchIncidents(PAGE_LIMIT, input.incidentsOffset, input.incidentFilters), { items: [], total: 0, limit: PAGE_LIMIT, offset: input.incidentsOffset })
    result.satellites = await optional(fetchSatellites, [])
    if (access.incidentsManage && access.users) result.users = await optional(fetchUsers, [])
    return result
  }

  if (activeTab === 'intel') {
    [result.spaceCVEs, result.assetAlerts] = await Promise.all([
      optional(() => fetchSpaceCVEs(PAGE_LIMIT, input.cvesOffset, input.cveFilters), { items: [], total: 0, limit: PAGE_LIMIT, offset: input.cvesOffset }),
      optional(() => fetchAssetAlerts(PAGE_LIMIT, input.alertsOffset), { items: [], total: 0, limit: PAGE_LIMIT, offset: input.alertsOffset }),
    ])
    return result
  }

  if (activeTab === 'connectivity' && access.connectivity) {
    [result.agentKeys, result.agentIdentities, result.satellites] = await Promise.all([optional(fetchAgentKeys, []), optional(fetchAgentIdentities, []), optional(fetchSatellites, [])])
    return result
  }

  if (activeTab === 'platform' && access.platform) {
    [result.organizations, result.platformUsers] = await Promise.all([optional(fetchOrganizations, []), optional(fetchPlatformUsers, [])])
    return result
  }

  if (activeTab === 'admin' && access.adminAccess) {
    const tasks: Promise<void>[] = []
    if (access.audit) tasks.push(optional(() => fetchAuditLog(PAGE_LIMIT, input.auditOffset), { items: [], total: 0, limit: PAGE_LIMIT, offset: input.auditOffset }).then((v) => { result.auditEntries = v }))
    if (access.users || access.admin) tasks.push(optional(fetchUsers, []).then((v) => { result.users = v }))
    if (access.rbac) {
      tasks.push(optional(fetchRBACPermissions, []).then((v) => { result.rbacPermissions = v }))
      tasks.push(optional(fetchRBACRoles, []).then((v) => { result.rbacRoles = v }))
    }
    await Promise.all(tasks)
  }
  return result
}
