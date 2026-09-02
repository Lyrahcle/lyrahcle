import { useEffect, useRef, useState, type ComponentType } from 'react'
import type { RouteTab } from '../lib/routeTabs'
import { buildDashboardNavigation } from './dashboard/navigation'
import type { RBACPermission, RBACRole, RBACUser } from '../types/security'
import Header from './Header.jsx'
import NavigationMenu from './NavigationMenu.jsx'
import SkipLink from './SkipLink'
import Panel from './Panel.tsx'
import Skeleton from './Skeleton.jsx'
import WorldMap from './WorldMap.jsx'
import ThreatIntelPanel from './ThreatIntelPanel.jsx'
import NewAssetForm from './NewAssetForm.jsx'
import NewSatelliteForm from './NewSatelliteForm.jsx'
import Footer from './Footer.jsx'
import ApiStatusBanner from './ApiStatusBanner.jsx'
import { AnomalyTimelineChart, CVESeverityChart, CVETimelineChart, IncidentSeverityChart, IncidentTimelineChart, OrbitalAltitudeChart, RFPowerChart, RFSpectrumChart } from './TelemetryCharts.jsx'
import { hasPermission } from '../lib/permissions'
import { useI18n } from '../lib/I18nContext'
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
  fetchRfReadings,
  fetchDetectionRules,
  fetchSatellites,
  fetchSpaceCVEs,
  fetchUsers,
  fetchPlatformAccess,
  fetchOrganizations,
  fetchPlatformUsers,
  fetchRBACPermissions,
  fetchRBACRoles,
  subscribeSatellites,
  subscribeIncidents,
  UnauthorizedError,
  ApiError,
  API_ERROR_EVENT,
} from '../lib/api'
import { lazyComponent } from '../lib/lazyComponent'

const AssetsTable = lazyComponent<any>(() => import('./AssetsTable.jsx'))
const RfPanel = lazyComponent<any>(() => import('./RfPanel.jsx'))
const AnomaliesPanel = lazyComponent<any>(() => import('./AnomaliesPanel.jsx'))
const DetectionRulesPanel = lazyComponent<any>(() => import('./DetectionRulesPanel.jsx'))
const SatellitesPanel = lazyComponent<any>(() => import('./SatellitesPanel.jsx'))
const OrbitalOperationsPanel = lazyComponent<any>(() => import('./OrbitalOperationsPanel.jsx'))
const MissionSecurityOperationsDashboard = lazyComponent<any>(() => import('./MissionSecurityOperationsDashboard.jsx'))
const MissionControl = lazyComponent<any>(() => import('./MissionControl.tsx'))
const MissionSecurityScore = lazyComponent<any>(() => import('./MissionSecurityScore.tsx'))
const SatelliteSecurityProfile = lazyComponent<any>(() => import('./SatelliteSecurityProfile.jsx'))
const GroundStationOperations = lazyComponent<any>(() => import('./GroundStationOperations.jsx'))
const AssetAlertsPanel = lazyComponent<any>(() => import('./AssetAlertsPanel.jsx'))
const IncidentsPanel = lazyComponent<any>(() => import('./IncidentsPanel.jsx'))
const Decoder = lazyComponent<any>(() => import('./Decoder.jsx'))
const TMTCSequenceAnalyzer = lazyComponent<any>(() => import('./TMTCSequenceAnalyzer.tsx'))
const ProgramManager = lazyComponent<any>(() => import('./ProgramManager.jsx'))
const ConnectivityPanel = lazyComponent<any>(() => import('./ConnectivityPanel.jsx'))
const AuditLogPanel = lazyComponent<any>(() => import('./AuditLogPanel.jsx'))
const UsersPanel = lazyComponent<any>(() => import('./UsersPanel.jsx'))
const SettingsPanel = lazyComponent<any>(() => import('./SettingsPanel.jsx'))
const PlatformOrganizationsPanel = lazyComponent<any>(() => import('./PlatformOrganizationsPanel.jsx'))
const PlatformOperationsOverview = lazyComponent<any>(() => import('./PlatformOperationsOverview.jsx'))
const SigningKeysPanel = lazyComponent<any>(() => import('./SigningKeysPanel.jsx'))
const RateLimitPolicyPanel = lazyComponent(() => import('./RateLimitPolicyPanel.tsx'))
const SecretProviderPanel = lazyComponent(() => import('./SecretProviderPanel.tsx'))
const IncidentResponsePanel = lazyComponent(() => import('./IncidentResponsePanel.tsx'))
const RBACPanel = lazyComponent(() => import('./RBACPanel.tsx'))
const RFHeatMap = lazyComponent<any>(() => import('./RFHeatMap.jsx'))
const RFSpectrumWaterfall = lazyComponent<any>(() => import('./RFSpectrumWaterfall.tsx'))
const RFAttackTriage = lazyComponent<any>(() => import('./RFAttackTriage.tsx'))
const AttackTimeline = lazyComponent<any>(() => import('./AttackTimeline.tsx'))
const SpaceThreatMatrix = lazyComponent<any>(() => import('./SpaceThreatMatrix.tsx'))
const CorrelationGraph = lazyComponent(() => import('./CorrelationGraph.tsx'))

const UntypedWorldMap = WorldMap as ComponentType<any>
const UntypedNewAssetForm = NewAssetForm as ComponentType<any>
const UntypedPlatformOperationsOverview = PlatformOperationsOverview as ComponentType<any>

const REFRESH_MS = 8000
const PAGE_LIMIT = 50

type DashboardEntity = Record<string, any>
type DashboardPage = {
  items: DashboardEntity[]
  total?: number
  limit?: number
  offset?: number
  [key: string]: any
}
type Status = 'checking' | 'online' | 'offline'
type CVEFilters = {
  severity: string
  keyword: string
  publishedFrom: string
  publishedTo: string
  sort: string
  order: string
}
type IncidentFilters = {
  status: string
  severity: string
  agentId: string
  satelliteId: string
}

interface DashboardProps {
  onLogout: () => void
  onSessionExpired: () => void
  activeTab: RouteTab
  onChangeTab: (tab: RouteTab) => void
}

export default function Dashboard({ onLogout, onSessionExpired, activeTab, onChangeTab }: DashboardProps) {
  const { t } = useI18n()
  const [assets, setAssets] = useState<DashboardPage | null>(null)
  const [programs, setPrograms] = useState<DashboardEntity[] | null>(null)
  const [rfReadings, setRfReadings] = useState<DashboardEntity[] | null>(null)
  const [rfAnomalies, setRfAnomalies] = useState<DashboardEntity[] | null>(null)
  const [detectionRules, setDetectionRules] = useState<DashboardPage | null>(null)
  const [satellites, setSatellites] = useState<DashboardEntity[] | null>(null)
  const [satellitesLive, setSatellitesLive] = useState(false)
  const [selectedSatelliteId, setSelectedSatelliteId] = useState<string | null>(null)
  const [tmtcAnalysis, setTmtcAnalysis] = useState<any | null>(null)
  const [spaceCVEs, setSpaceCVEs] = useState<DashboardPage | null>(null)
  const [assetAlerts, setAssetAlerts] = useState<DashboardPage | null>(null)
  const [incidents, setIncidents] = useState<DashboardPage | null>(null)
  const [agentKeys, setAgentKeys] = useState<DashboardEntity[] | null>(null)
  const [agentIdentities, setAgentIdentities] = useState<DashboardEntity[] | null>(null)
  const [auditEntries, setAuditEntries] = useState<DashboardPage | null>(null)
  const [users, setUsers] = useState<RBACUser[] | null>(null)
  const [platformAdmin, setPlatformAdmin] = useState(false)
  const [organizations, setOrganizations] = useState<DashboardEntity[] | null>(null)
  const [platformUsers, setPlatformUsers] = useState<DashboardEntity[] | null>(null)
  const [rbacPermissions, setRbacPermissions] = useState<RBACPermission[] | null>(null)
  const [rbacRoles, setRbacRoles] = useState<RBACRole[] | null>(null)
  const [backendStatus, setBackendStatus] = useState<Status>('checking')
  const [dbStatus, setDbStatus] = useState<Status>('checking')
  const [apiIssue, setApiIssue] = useState<ApiError | null>(null)
  const [retryNonce, setRetryNonce] = useState(0)
  const [mobileNavigationOpen, setMobileNavigationOpen] = useState(false)
  const [navigationCollapsed, setNavigationCollapsed] = useState(() => localStorage.getItem('lyra_nav_collapsed') === '1')
  const rateLimitedUntilRef = useRef(0)

  useEffect(() => {
    localStorage.setItem('lyra_nav_collapsed', navigationCollapsed ? '1' : '0')
  }, [navigationCollapsed])

  useEffect(() => {
    setMobileNavigationOpen(false)
  }, [activeTab])

  const [assetsOffset, setAssetsOffset] = useState(0)
  const [cvesOffset, setCvesOffset] = useState(0)
  const [cveFilters, setCveFilters] = useState<CVEFilters>({ severity: '', keyword: '', publishedFrom: '', publishedTo: '', sort: 'published', order: 'desc' })
  const cveFiltersRef = useRef(cveFilters)
  useEffect(() => {
    cveFiltersRef.current = cveFilters
  }, [cveFilters])
  const [alertsOffset, setAlertsOffset] = useState(0)
  const [incidentsOffset, setIncidentsOffset] = useState(0)
  const [incidentFilters, setIncidentFilters] = useState<IncidentFilters>({ status: '', severity: '', agentId: '', satelliteId: '' })
  const [auditOffset, setAuditOffset] = useState(0)
  const assetsOffsetRef = useRef(0)
  const cvesOffsetRef = useRef(0)
  const alertsOffsetRef = useRef(0)
  const incidentsOffsetRef = useRef(0)
  const incidentFiltersRef = useRef(incidentFilters)
  const auditOffsetRef = useRef(0)
  useEffect(() => {
    assetsOffsetRef.current = assetsOffset
  }, [assetsOffset])
  useEffect(() => {
    cvesOffsetRef.current = cvesOffset
  }, [cvesOffset])
  useEffect(() => {
    alertsOffsetRef.current = alertsOffset
  }, [alertsOffset])
  useEffect(() => {
    auditOffsetRef.current = auditOffset
  }, [auditOffset])
  useEffect(() => {
    incidentsOffsetRef.current = incidentsOffset
  }, [incidentsOffset])
  useEffect(() => {
    incidentFiltersRef.current = incidentFilters
  }, [incidentFilters])


  useEffect(() => {
    function handleApiError(event) {
      const issue = event.detail
      if (!(issue instanceof ApiError)) return
      if (issue.kind === 'rate_limited' && issue.retryAfterSeconds > 0) {
        rateLimitedUntilRef.current = Date.now() + issue.retryAfterSeconds * 1000
      }
      setApiIssue(issue)
    }
    window.addEventListener(API_ERROR_EVENT, handleApiError)
    return () => window.removeEventListener(API_ERROR_EVENT, handleApiError)
  }, [])

  function retryFailedRequests() {
    if (Date.now() < rateLimitedUntilRef.current) return
    setApiIssue(null)
    setRetryNonce((value) => value + 1)
  }

  const admin = hasPermission('users.manage')
  const programsAccess = hasPermission('programs.read') || hasPermission('programs.manage')
  const assetsAccess = hasPermission('assets.read') || hasPermission('assets.manage')
  const usersRead = hasPermission('users.read')
  const incidentsRead = hasPermission('incidents.read')
  const incidentsManage = hasPermission('incidents.manage')
  const rfRead = hasPermission('rf.read')
  const alertsRead = hasPermission('alerts.read')
  const agentsRead = hasPermission('agents.read')
  const agentsManage = hasPermission('agents.manage')
  const auditAccess = hasPermission('audit.read')
  const rbacRead = hasPermission('rbac.read')
  const rbacManage = hasPermission('rbac.manage')
  const adminAccess = admin || usersRead || auditAccess || rbacRead || rbacManage
  const connectivityAccess = agentsRead || agentsManage
  const orbitalAccess = hasPermission('orbital.read') || hasPermission('orbital.manage')
  const tabs = buildDashboardNavigation(t, {
    connectivity: connectivityAccess,
    administration: adminAccess,
    platform: platformAdmin,
  })
  const activeNavigationItem = tabs.find((item) => item.id === activeTab) ?? tabs[0]

  function handleCommandNavigate(tab: RouteTab, result?: { type?: string; id?: string }) {
    if (result?.type === 'satellites' && result.id) {
      setSelectedSatelliteId(result.id)
      onChangeTab('orbital')
      return
    }
    setSelectedSatelliteId(null)
    onChangeTab(tab)
  }

  useEffect(() => {
    let cancelled = false
    fetchPlatformAccess()
      .then((result) => {
        if (!cancelled) setPlatformAdmin(Boolean(result?.platform_admin))
      })
      .catch((err) => {
        if (err instanceof UnauthorizedError) onSessionExpired()
        else if (!cancelled) setPlatformAdmin(false)
      })
    return () => { cancelled = true }
  }, [onSessionExpired])

  useEffect(() => {
    let cancelled = false

    async function safeFetch(fetcher: () => Promise<any>, setter: (value: any) => void, fallback: any = []) {
      try {
        const data = await fetcher()
        if (!cancelled) setter(data ?? fallback)
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          onSessionExpired()
          return
        }
        if (!cancelled) setter((prev) => prev ?? fallback)
      }
    }

    async function poll() {
      if (Date.now() < rateLimitedUntilRef.current) return
      try {
        await fetchPing()
        if (!cancelled) {
          setBackendStatus('online')
          setApiIssue((current) => current?.kind === 'network' ? null : current)
        }
      } catch {
        if (!cancelled) setBackendStatus('offline')
      }

      if (activeTab === 'mission') {
        const requests = [safeFetch(fetchSatellites, setSatellites)]
        if (rfRead) {
          requests.push(safeFetch(fetchRfReadings, setRfReadings))
          requests.push(safeFetch(fetchRfAnomalies, setRfAnomalies))
        }
        if (incidentsRead) requests.push(safeFetch(() => fetchIncidents(PAGE_LIMIT, 0, {}), setIncidents, { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
        if (connectivityAccess) requests.push(safeFetch(fetchAgentIdentities, setAgentIdentities))
        if (alertsRead) requests.push(safeFetch(() => fetchAssetAlerts(PAGE_LIMIT, 0), setAssetAlerts, { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
        await Promise.all(requests)
        return
      }

      if (activeTab === 'orbital') {
        const requests = [safeFetch(fetchSatellites, setSatellites)]
        if (incidentsRead) {
          requests.push(safeFetch(
            () => fetchIncidents(PAGE_LIMIT, 0, {}),
            setIncidents,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ))
        }
        if (rfRead) requests.push(safeFetch(fetchRfAnomalies, setRfAnomalies))
        if (connectivityAccess) requests.push(safeFetch(fetchAgentIdentities, setAgentIdentities))
        if (alertsRead) {
          requests.push(safeFetch(
            () => fetchAssetAlerts(PAGE_LIMIT, 0),
            setAssetAlerts,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ))
        }
        await Promise.all(requests)
        return
      }

      if (activeTab === 'graph') {
        const requests: Promise<void>[] = []
        if (orbitalAccess) requests.push(safeFetch(fetchSatellites, setSatellites))
        if (connectivityAccess) requests.push(safeFetch(fetchAgentIdentities, setAgentIdentities))
        if (rfRead) {
          requests.push(safeFetch(fetchRfReadings, setRfReadings))
          requests.push(safeFetch(fetchRfAnomalies, setRfAnomalies))
        }
        if (incidentsRead) {
          requests.push(safeFetch(
            () => fetchIncidents(PAGE_LIMIT, 0, {}),
            setIncidents,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ))
        }
        if (alertsRead) {
          requests.push(safeFetch(
            () => fetchAssetAlerts(PAGE_LIMIT, 0),
            setAssetAlerts,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ))
        }
        if (assetsAccess) requests.push(safeFetch(() => fetchAssets(PAGE_LIMIT, 0), setAssets, { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
        if (programsAccess) requests.push(safeFetch(fetchPrograms, setPrograms))
        requests.push(safeFetch(() => fetchSpaceCVEs(PAGE_LIMIT, 0, { severity: '', keyword: '', publishedFrom: '', publishedTo: '', sort: 'published', order: 'desc' }), setSpaceCVEs, { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }))
        await Promise.all(requests)
        return
      }

      if (activeTab === 'assets') {
        try {
          const [assetData, programData] = await Promise.all([
            fetchAssets(PAGE_LIMIT, assetsOffsetRef.current),
            programsAccess ? fetchPrograms() : Promise.resolve([]),
          ])
          if (!cancelled) {
            setAssets(assetData ?? { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
            setPrograms(programData ?? [])
            setDbStatus('online')
          }
        } catch (err) {
          if (err instanceof UnauthorizedError) {
            onSessionExpired()
            return
          }
          if (!cancelled) {
            setAssets((prev) => prev ?? { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
            setPrograms((prev) => prev ?? [])
            setDbStatus('offline')
          }
        }
        return
      }

      if (activeTab === 'stations' && connectivityAccess) {
        const requests = [safeFetch(fetchAgentIdentities, setAgentIdentities)]
        if (orbitalAccess) requests.push(safeFetch(fetchSatellites, setSatellites))
        if (rfRead) {
          requests.push(safeFetch(fetchRfReadings, setRfReadings))
          requests.push(safeFetch(fetchRfAnomalies, setRfAnomalies))
        }
        if (incidentsRead) {
          requests.push(safeFetch(
            () => fetchIncidents(PAGE_LIMIT, 0, {}),
            setIncidents,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ))
        }
        await Promise.all(requests)
        return
      }

      if (activeTab === 'signal') {
        await Promise.all([
          safeFetch(fetchRfReadings, setRfReadings),
          safeFetch(fetchRfAnomalies, setRfAnomalies),
          safeFetch(fetchAgentIdentities, setAgentIdentities),
          safeFetch(() => fetchDetectionRules(PAGE_LIMIT, 0), setDetectionRules, { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 }),
        ])
        return
      }

      if (activeTab === 'incidents') {
        const requests = [
          safeFetch(
            () => fetchIncidents(PAGE_LIMIT, incidentsOffsetRef.current, incidentFiltersRef.current),
            setIncidents,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ),
          safeFetch(fetchSatellites, setSatellites),
        ]
        if (incidentsManage && usersRead) requests.push(safeFetch(fetchUsers, setUsers))
        await Promise.all(requests)
        return
      }

      if (activeTab === 'intel') {
        await Promise.all([
          safeFetch(
            () => fetchSpaceCVEs(PAGE_LIMIT, cvesOffsetRef.current, cveFiltersRef.current),
            setSpaceCVEs,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ),
          safeFetch(
            () => fetchAssetAlerts(PAGE_LIMIT, alertsOffsetRef.current),
            setAssetAlerts,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ),
        ])
        return
      }

      if (activeTab === 'connectivity' && connectivityAccess) {
        await Promise.all([
          safeFetch(fetchAgentKeys, setAgentKeys),
          safeFetch(fetchAgentIdentities, setAgentIdentities),
          safeFetch(fetchSatellites, setSatellites),
        ])
        return
      }

      if (activeTab === 'platform' && platformAdmin) {
        await Promise.all([safeFetch(fetchOrganizations, setOrganizations), safeFetch(fetchPlatformUsers, setPlatformUsers)])
        return
      }

      if (activeTab === 'admin' && adminAccess) {
        const requests: Promise<void>[] = []
        if (auditAccess) {
          requests.push(safeFetch(
            () => fetchAuditLog(PAGE_LIMIT, auditOffsetRef.current),
            setAuditEntries,
            { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
          ))
        }
        if (usersRead || admin) requests.push(safeFetch(fetchUsers, setUsers))
        if (rbacRead) {
          requests.push(safeFetch(fetchRBACPermissions, setRbacPermissions))
          requests.push(safeFetch(fetchRBACRoles, setRbacRoles))
        }
        await Promise.all(requests)
      }
    }

    poll()
    const id = setInterval(poll, REFRESH_MS)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [onSessionExpired, activeTab, admin, auditAccess, assetsOffset, cvesOffset, cveFilters, auditOffset, incidentsOffset, incidentFilters, platformAdmin, adminAccess, usersRead, rbacRead, programsAccess, assetsAccess, incidentsRead, rfRead, alertsRead, connectivityAccess, orbitalAccess, retryNonce])

  useEffect(() => {
    if (activeTab !== 'orbital') {
      setSatellitesLive(false)
      setSelectedSatelliteId(null)
      return undefined
    }
    const unsubscribe = subscribeSatellites({
      onData: (data) => {
        setSatellites(data)
        setSatellitesLive(true)
      },
      onError: (err) => {
        setSatellitesLive(false)
        if (err instanceof UnauthorizedError) onSessionExpired()
      },
    })
    return unsubscribe
  }, [activeTab])

  useEffect(() => {
    if (activeTab !== 'incidents') return undefined
    const unsubscribe = subscribeIncidents({
      onData: (data) => {
        if (incidentsOffsetRef.current === 0 && !incidentFiltersRef.current.status && !incidentFiltersRef.current.severity && !incidentFiltersRef.current.agentId && !incidentFiltersRef.current.satelliteId) {
          setIncidents(data)
        }
      },
      onError: (err) => {
        if (err instanceof UnauthorizedError) onSessionExpired()
      },
    })
    return unsubscribe
  }, [activeTab])

  const lastRf = rfReadings?.[0]
  const sdrStatus =
    lastRf && Date.now() - new Date(lastRf.captured_at).getTime() < 30000 ? 'live' : 'idle'

  async function reloadAssets() {
    setAssetsOffset(0)
    try {
      const [assetData, programData] = await Promise.all([fetchAssets(PAGE_LIMIT, 0), programsAccess ? fetchPrograms() : Promise.resolve([])])
      setAssets(assetData)
      setPrograms(programData ?? [])
    } catch {
    }
  }
  function handleCveFiltersChange(next: CVEFilters) {
    setCvesOffset(0)
    setCveFilters(next)
  }
  function reloadDetectionRules() {
    fetchDetectionRules(PAGE_LIMIT, 0).then(setDetectionRules).catch(() => {})
  }
  function reloadSatellites() {
    fetchSatellites().then(setSatellites).catch(() => {})
  }
  function reloadAgentKeys() {
    fetchAgentKeys().then(setAgentKeys).catch(() => {})
  }
  function reloadAgentIdentities() {
    fetchAgentIdentities().then(setAgentIdentities).catch(() => {})
  }
  function reloadUsers() {
    fetchUsers().then(setUsers).catch(() => {})
  }
  function reloadOrganizations() {
    fetchOrganizations().then(setOrganizations).catch(() => {})
  }
  function reloadPlatformUsers() {
    fetchPlatformUsers().then(setPlatformUsers).catch(() => {})
  }
  function reloadRBAC() {
    Promise.all([fetchRBACPermissions(), fetchRBACRoles()])
      .then(([permissions, roles]) => {
        setRbacPermissions(permissions)
        setRbacRoles(roles)
      })
      .catch(() => {})
  }
  function reloadAssetAlerts() {
    fetchAssetAlerts(PAGE_LIMIT, alertsOffsetRef.current).then(setAssetAlerts).catch(() => {})
  }
  function reloadIncidents() {
    fetchIncidents(PAGE_LIMIT, incidentsOffsetRef.current, incidentFiltersRef.current).then(setIncidents).catch(() => {})
  }
  function handleIncidentFiltersChange(next: IncidentFilters) {
    setIncidentsOffset(0)
    setIncidentFilters(next)
  }

  const selectedSatellite = selectedSatelliteId
    ? (satellites ?? []).find((satellite) => satellite.id === selectedSatelliteId) ?? null
    : null

  return (
    <div className="min-h-screen bg-transparent text-lyra-text">
      <SkipLink />
      <NavigationMenu
        items={tabs}
        active={activeTab}
        onChange={onChangeTab}
        collapsed={navigationCollapsed}
        onToggleCollapse={() => setNavigationCollapsed((value) => !value)}
        mobileOpen={mobileNavigationOpen}
        onCloseMobile={() => setMobileNavigationOpen(false)}
      />
      <div className={`min-h-screen transition-[padding] duration-200 ${navigationCollapsed ? 'lg:pl-20' : 'lg:pl-64'}`}>
        <Header
          onLogout={onLogout}
          onHome={() => onChangeTab('orbital')}
          onOpenSettings={() => onChangeTab('settings')}
          onOpenNavigation={() => setMobileNavigationOpen(true)}
          activePageLabel={activeNavigationItem?.label ?? t('navigation.items.overview')}
          backendStatus={backendStatus}
          dbStatus={dbStatus}
          sdrStatus={sdrStatus}
          navigationItems={tabs}
          onCommandNavigate={handleCommandNavigate}
        />
        <ApiStatusBanner
          issue={apiIssue}
          onRetry={apiIssue?.retryable ? retryFailedRequests : undefined}
          onDismiss={() => setApiIssue(null)}
        />

        <main id="main-content" className="mx-auto max-w-7xl px-4 py-8 sm:px-6 sm:py-10 xl:px-8">
        {activeTab === 'orbital' && (
          selectedSatellite ? (
            <SatelliteSecurityProfile
              satellite={selectedSatellite}
              incidents={incidents}
              anomalies={rfAnomalies}
              agentIdentities={agentIdentities}
              assetAlerts={assetAlerts}
              access={{ incidents: incidentsRead, rf: rfRead, agents: connectivityAccess, alerts: alertsRead }}
              onBack={() => setSelectedSatelliteId(null)}
              onNavigate={(tab) => { setSelectedSatelliteId(null); onChangeTab(tab) }}
            />
          ) : (
          <div className="grid gap-6">
            {satellites === null ? <Skeleton rows={2} /> : (
              <MissionSecurityOperationsDashboard
                satellites={satellites}
                incidents={incidents}
                anomalies={rfAnomalies}
                agentIdentities={agentIdentities}
                assetAlerts={assetAlerts}
                live={satellitesLive}
                backendStatus={backendStatus}
                access={{ incidents: incidentsRead, rf: rfRead, agents: connectivityAccess, alerts: alertsRead }}
                onNavigate={onChangeTab}
              />
            )}
            <div id="orbital-ground-track">
            <Panel
              eyebrow={t('panels.groundTrack')}
              title={t('panels.liveOrbitalPositions')}
              bleed
              titleExtra={
                satellitesLive && (
                  <span className="flex items-center gap-1.5 rounded-full border border-lyra-good/30 bg-lyra-good/10 px-2 py-0.5 font-mono text-[10px] tracking-wider text-lyra-good">
                    <span className="h-1.5 w-1.5 rounded-full bg-lyra-good motion-safe:animate-pulse" />
                    {t('panels.live')}
                  </span>
                )
              }
            >
              {satellites === null ? (
                <div className="px-7 pb-7">
                  <Skeleton rows={1} />
                </div>
              ) : (
                <div className="px-2 pb-4 sm:px-3">
                  <UntypedWorldMap satellites={satellites} />
                </div>
              )}
            </Panel>
            </div>
            <Panel eyebrow={t('panels.orbitalTracking')} title="Orbital altitude profile">
              {satellites === null ? <Skeleton /> : <OrbitalAltitudeChart satellites={satellites} />}
            </Panel>
            <div id="orbital-management">
            <Panel eyebrow={t('panels.orbitalTracking')} title={t('panels.trackedSatellites')}>
              {hasPermission('orbital.manage') && <NewSatelliteForm onCreated={reloadSatellites} />}
              {satellites === null ? <Skeleton /> : <SatellitesPanel satellites={satellites} incidents={incidents} anomalies={rfAnomalies} assetAlerts={assetAlerts} access={{ incidents: incidentsRead, rf: rfRead, alerts: alertsRead }} canManage={hasPermission('orbital.manage')} onChanged={reloadSatellites} onOpenProfile={(satellite) => setSelectedSatelliteId(satellite.id)} />}
            </Panel>
            </div>
            {(hasPermission('orbital.read') || hasPermission('orbital.manage')) && (
              <div id="orbital-operations">
              <Panel eyebrow={t('panels.orbitalOperations')} title={t('orbitalOperations.title')}>
                {satellites === null ? <Skeleton /> : <OrbitalOperationsPanel satellites={satellites} />}
              </Panel>
              </div>
            )}
          </div>
          )
        )}

        {activeTab === 'mission' && (
          satellites === null ? <Skeleton rows={4} /> : (
            <div className="grid gap-6">
            <MissionSecurityScore
              satellites={satellites}
              incidents={incidents?.items ?? []}
              anomalies={rfAnomalies ?? []}
              agents={agentIdentities ?? []}
              alerts={assetAlerts?.items ?? []}
              backendStatus={backendStatus}
              access={{ incidents: incidentsRead, rf: rfRead, agents: connectivityAccess, alerts: alertsRead }}
            />
            <MissionControl
              satellites={satellites}
              readings={rfReadings ?? []}
              anomalies={rfAnomalies ?? []}
              incidents={incidents}
              agents={agentIdentities ?? []}
              assetAlerts={assetAlerts}
              access={{ orbital: orbitalAccess, rf: rfRead, incidents: incidentsRead, incidentsManage, agents: connectivityAccess, alerts: alertsRead }}
              onNavigate={onChangeTab}
              tmtcAnalysis={tmtcAnalysis}
              onIncidentCreated={reloadIncidents}
            />
            </div>
          )
        )}

        {activeTab === 'graph' && (
          <CorrelationGraph
            satellites={satellites}
            agents={agentIdentities}
            readings={rfReadings}
            anomalies={rfAnomalies}
            incidents={incidents}
            assets={assets}
            programs={programs}
            alerts={assetAlerts}
            cves={spaceCVEs}
            access={{ orbital: orbitalAccess, agents: connectivityAccess, rf: rfRead, incidents: incidentsRead, assets: assetsAccess, programs: programsAccess, alerts: alertsRead }}
            onNavigate={onChangeTab}
            onOpenSatellite={(id) => { setSelectedSatelliteId(id); onChangeTab('orbital') }}
          />
        )}

        {activeTab === 'stations' && connectivityAccess && (
          agentIdentities === null ? <Skeleton rows={3} /> : (
            <GroundStationOperations
              agents={agentIdentities}
              satellites={satellites ?? []}
              readings={rfReadings ?? []}
              anomalies={rfAnomalies ?? []}
              incidents={incidents?.items ?? []}
              access={{ orbital: orbitalAccess, rf: rfRead, incidents: incidentsRead }}
              onNavigate={onChangeTab}
            />
          )
        )}

        {activeTab === 'assets' && (
          <Panel eyebrow={t('panels.assetInventory')} title={t('panels.trackedAssets')}>
            {hasPermission('assets.manage') && programsAccess && (
              <UntypedNewAssetForm
                onCreated={reloadAssets}
                programs={programs ?? []}
                onProgramsChanged={reloadAssets}
              />
            )}
            {hasPermission('assets.manage') && !programsAccess && (
              <p className="mb-4 text-xs text-lyra-mutedDim">{t('assetInventory.programAccessRequired')}</p>
            )}
            {assets === null || programs === null ? (
              <Skeleton />
            ) : (
              <>
                <AssetsTable
                  page={assets}
                  programs={programs}
                  onChanged={reloadAssets}
                  onPrev={() => setAssetsOffset((o) => Math.max(0, o - PAGE_LIMIT))}
                  onNext={() => setAssetsOffset((o) => o + PAGE_LIMIT)}
                />
                {(hasPermission('programs.read') || hasPermission('programs.manage')) && (
                  <ProgramManager programs={programs} onChanged={reloadAssets} />
                )}
              </>
            )}
          </Panel>
        )}

        {activeTab === 'signal' && (
          <div className="grid gap-6">
            <Panel eyebrow={t('panels.signalIntelligence')} title={t('panels.rfSpectrum')}>
              {rfReadings === null ? <Skeleton /> : (
                <div className="grid gap-6">
                  <div className="grid gap-6 lg:grid-cols-2">
                    <RFPowerChart readings={rfReadings} />
                    <RFSpectrumChart readings={rfReadings} />
                  </div>
                  <RFSpectrumWaterfall readings={rfReadings} />
                  <RfPanel readings={rfReadings} />
                </div>
              )}
            </Panel>
            <Panel eyebrow={t('panels.signalIntelligence')} title={t('rfHeatMap.panelTitle')}>
              {rfReadings === null || agentIdentities === null ? <Skeleton /> : (
                <RFHeatMap agents={agentIdentities} readings={rfReadings} />
              )}
            </Panel>
            <Panel eyebrow={t('panels.anomalyDetection')} title={t('rfAttackTriage.panelTitle')}>
              {rfAnomalies === null ? <Skeleton /> : <RFAttackTriage anomalies={rfAnomalies} />}
            </Panel>
            <Panel eyebrow={t('panels.anomalyDetection')} title={t('attackTimeline.panelTitle')}>
              {rfReadings === null || rfAnomalies === null || (incidentsRead && incidents === null) ? <Skeleton /> : (
                <AttackTimeline readings={rfReadings ?? []} anomalies={rfAnomalies ?? []} incidents={incidents?.items ?? []} />
              )}
            </Panel>
            <Panel eyebrow={t('panels.anomalyDetection')} title={t('spaceThreatMatrix.panelTitle')}>
              {rfAnomalies === null || (incidentsRead && incidents === null) ? <Skeleton /> : (
                <SpaceThreatMatrix anomalies={rfAnomalies ?? []} incidents={incidents?.items ?? []} />
              )}
            </Panel>
            <Panel eyebrow={t('panels.anomalyDetection')} title={t('panels.flaggedAnomalies')}>
              {rfAnomalies === null ? <Skeleton /> : (
                <div className="grid gap-6">
                  <AnomalyTimelineChart anomalies={rfAnomalies} />
                  <AnomaliesPanel anomalies={rfAnomalies} />
                </div>
              )}
            </Panel>
            <Panel eyebrow={t('panels.detectionEngineering')} title={t('panels.rfDetectionRules')}>
              {detectionRules === null ? <Skeleton /> : <DetectionRulesPanel page={detectionRules} onChanged={reloadDetectionRules} />}
            </Panel>
          </div>
        )}

        {activeTab === 'incidents' && (
          <div className="grid gap-6">
            <Panel eyebrow={t('panels.incidentResponse')} title="Incident overview">
              {incidents === null ? <Skeleton /> : (
                <div className="grid gap-6 lg:grid-cols-2">
                  <IncidentSeverityChart incidents={incidents.items ?? []} />
                  <IncidentTimelineChart incidents={incidents.items ?? []} />
                </div>
              )}
            </Panel>
            <Panel eyebrow={t('panels.incidentResponse')} title={t('panels.securityIncidents')}>
              {incidents === null ? (
                <Skeleton />
              ) : (
                <IncidentsPanel
                  page={incidents}
                  filters={incidentFilters}
                  onFiltersChange={handleIncidentFiltersChange}
                  onPrev={() => setIncidentsOffset((o) => Math.max(0, o - PAGE_LIMIT))}
                  onNext={() => setIncidentsOffset((o) => o + PAGE_LIMIT)}
                  onChanged={reloadIncidents}
                  satellites={satellites ?? []}
                  users={users ?? []}
                />
              )}
            </Panel>
          </div>
        )}

        {activeTab === 'intel' && (
          <div className="grid gap-6">
            <Panel eyebrow={t('panels.threatIntelligence')} title="CVE overview">
              {spaceCVEs === null ? <Skeleton /> : (
                <div className="grid gap-6 lg:grid-cols-2">
                  <CVESeverityChart cves={spaceCVEs.items ?? []} />
                  <CVETimelineChart cves={spaceCVEs.items ?? []} />
                </div>
              )}
            </Panel>
            <Panel eyebrow={t('panels.threatIntelligence')} title={t('panels.spaceCves')}>
              <ThreatIntelPanel
                page={spaceCVEs}
                loading={spaceCVEs === null}
                filters={cveFilters}
                onFiltersChange={handleCveFiltersChange}
                onPrev={() => setCvesOffset((o) => Math.max(0, o - PAGE_LIMIT))}
                onNext={() => setCvesOffset((o) => o + PAGE_LIMIT)}
              />
            </Panel>
            <Panel eyebrow={t('panels.assetAlerts')} title={t('panels.matchedVulnerabilities')}>
              {assetAlerts === null ? (
                <Skeleton />
              ) : (
                <AssetAlertsPanel
                  page={assetAlerts}
                  onPrev={() => setAlertsOffset((o) => Math.max(0, o - PAGE_LIMIT))}
                  onNext={() => setAlertsOffset((o) => o + PAGE_LIMIT)}
                  onChanged={reloadAssetAlerts}
                />
              )}
            </Panel>
          </div>
        )}

        {activeTab === 'decoder' && (
          <div className="grid gap-6">
            <Panel eyebrow={t('panels.tmtc')} title={t('panels.protocolDecoder')}>
              <Decoder />
            </Panel>
            <Panel eyebrow={t('panels.tmtc')} title="CCSDS sequence security">
              <TMTCSequenceAnalyzer onAnalysis={setTmtcAnalysis} />
            </Panel>
          </div>
        )}

        {activeTab === 'connectivity' && (
          <ConnectivityPanel
            agentKeys={agentKeys}
            agentIdentities={agentIdentities}
            satellites={satellites ?? []}
            onAgentKeysChanged={reloadAgentKeys}
            onAgentIdentitiesChanged={reloadAgentIdentities}
          />
        )}

                {activeTab === 'platform' && platformAdmin && (
          <div className="grid gap-6">
            <UntypedPlatformOperationsOverview organizations={organizations ?? []} platformUsers={platformUsers ?? []} />
            <Panel eyebrow={t('platform.eyebrow')} title={t('platform.title')}>
              {organizations === null || platformUsers === null ? <Skeleton /> : <PlatformOrganizationsPanel organizations={organizations} platformUsers={platformUsers} onChanged={reloadOrganizations} onPlatformUsersChanged={reloadPlatformUsers} />}
            </Panel>
            <Panel eyebrow={t('signingKeys.eyebrow')} title={t('signingKeys.title')}>
              <SigningKeysPanel />
            </Panel>
            <Panel eyebrow="Secret management" title="Secret provider">
              <SecretProviderPanel />
            </Panel>
            <Panel eyebrow={t('rateLimits.eyebrow')} title={t('rateLimits.title')}>
              <RateLimitPolicyPanel />
            </Panel>
            <Panel eyebrow="Security operations" title="Incident response">
              <IncidentResponsePanel />
            </Panel>
          </div>
        )}

        {activeTab === 'admin' && adminAccess && (
          <div className="grid gap-6">
            {admin && (
              <Panel eyebrow={t('panels.accessControl')} title={t('panels.usersAndRoles')}>
                {users === null ? <Skeleton /> : <UsersPanel users={users} onChanged={reloadUsers} />}
              </Panel>
            )}
            {rbacRead && (
              <Panel eyebrow={t('panels.accessControl')} title={t('rbac.title')}>
                {rbacPermissions === null || rbacRoles === null || ((usersRead || admin) && users === null) ? (
                  <Skeleton />
                ) : (
                  <RBACPanel permissions={rbacPermissions} roles={rbacRoles} users={users ?? []} canReadUsers={usersRead || admin} onChanged={reloadRBAC} />
                )}
              </Panel>
            )}
            {auditAccess && (
              <Panel eyebrow={t('panels.security')} title={t('panels.auditLog')}>
                {auditEntries === null ? (
                  <Skeleton />
                ) : (
                  <AuditLogPanel
                    page={auditEntries}
                    onPrev={() => setAuditOffset((o) => Math.max(0, o - PAGE_LIMIT))}
                    onNext={() => setAuditOffset((o) => o + PAGE_LIMIT)}
                  />
                )}
              </Panel>
            )}
          </div>
        )}

        {activeTab === 'settings' && (
          <Panel eyebrow={t('settings.eyebrow')} title={t('settings.title')}>
            <SettingsPanel admin={admin} onNavigateToConnectivity={() => onChangeTab('connectivity')} onSecurityChange={onLogout} />
          </Panel>
        )}
        </main>

        <Footer />
      </div>
    </div>
  )
}
