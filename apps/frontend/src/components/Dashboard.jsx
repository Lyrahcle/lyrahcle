import { useEffect, useRef, useState } from 'react'
import Header from './Header.jsx'
import Panel from './Panel.jsx'
import Tabs from './Tabs.jsx'
import Skeleton from './Skeleton.jsx'
import AssetsTable from './AssetsTable.jsx'
import RfPanel from './RfPanel.jsx'
import AnomaliesPanel from './AnomaliesPanel.jsx'
import DetectionRulesPanel from './DetectionRulesPanel.jsx'
import SatellitesPanel from './SatellitesPanel.jsx'
import OrbitalOperationsPanel from './OrbitalOperationsPanel.jsx'
import WorldMap from './WorldMap.jsx'
import ThreatIntelPanel from './ThreatIntelPanel.jsx'
import AssetAlertsPanel from './AssetAlertsPanel.jsx'
import IncidentsPanel from './IncidentsPanel.jsx'
import Decoder from './Decoder.jsx'
import NewAssetForm from './NewAssetForm.jsx'
import ProgramManager from './ProgramManager.jsx'
import NewSatelliteForm from './NewSatelliteForm.jsx'
import ConnectivityPanel from './ConnectivityPanel.jsx'
import AuditLogPanel from './AuditLogPanel.jsx'
import UsersPanel from './UsersPanel.jsx'
import SettingsPanel from './SettingsPanel.jsx'
import PlatformOrganizationsPanel from './PlatformOrganizationsPanel.jsx'
import SigningKeysPanel from './SigningKeysPanel.jsx'
import RateLimitPolicyPanel from './RateLimitPolicyPanel.jsx'
import RBACPanel from './RBACPanel.jsx'
import Footer from './Footer.jsx'
import ApiStatusBanner from './ApiStatusBanner.jsx'
import { AnomalyTimelineChart, CVESeverityChart, CVETimelineChart, IncidentSeverityChart, IncidentTimelineChart, OrbitalAltitudeChart, RFPowerChart, RFSpectrumChart, RFStationMapChart } from './TelemetryCharts.jsx'
import { hasPermission } from '../lib/permissions.js'
import { useI18n } from '../lib/I18nContext.jsx'
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
} from '../lib/api.js'

const REFRESH_MS = 8000
const PAGE_LIMIT = 50

export default function Dashboard({ onLogout, onSessionExpired, activeTab, onChangeTab }) {
  const { t } = useI18n()
  const [assets, setAssets] = useState(null)
  const [programs, setPrograms] = useState(null)
  const [rfReadings, setRfReadings] = useState(null)
  const [rfAnomalies, setRfAnomalies] = useState(null)
  const [detectionRules, setDetectionRules] = useState(null)
  const [satellites, setSatellites] = useState(null)
  const [satellitesLive, setSatellitesLive] = useState(false)
  const [spaceCVEs, setSpaceCVEs] = useState(null)
  const [assetAlerts, setAssetAlerts] = useState(null)
  const [incidents, setIncidents] = useState(null)
  const [agentKeys, setAgentKeys] = useState(null)
  const [agentIdentities, setAgentIdentities] = useState(null)
  const [auditEntries, setAuditEntries] = useState(null)
  const [users, setUsers] = useState(null)
  const [platformAdmin, setPlatformAdmin] = useState(false)
  const [organizations, setOrganizations] = useState(null)
  const [platformUsers, setPlatformUsers] = useState(null)
  const [rbacPermissions, setRbacPermissions] = useState(null)
  const [rbacRoles, setRbacRoles] = useState(null)
  const [backendStatus, setBackendStatus] = useState('checking')
  const [dbStatus, setDbStatus] = useState('checking')
  const [apiIssue, setApiIssue] = useState(null)
  const [retryNonce, setRetryNonce] = useState(0)
  const rateLimitedUntilRef = useRef(0)

  const [assetsOffset, setAssetsOffset] = useState(0)
  const [cvesOffset, setCvesOffset] = useState(0)
  const [cveFilters, setCveFilters] = useState({ severity: '', keyword: '', publishedFrom: '', publishedTo: '', sort: 'published', order: 'desc' })
  const cveFiltersRef = useRef(cveFilters)
  useEffect(() => {
    cveFiltersRef.current = cveFilters
  }, [cveFilters])
  const [alertsOffset, setAlertsOffset] = useState(0)
  const [incidentsOffset, setIncidentsOffset] = useState(0)
  const [incidentFilters, setIncidentFilters] = useState({ status: '', severity: '', agentId: '', satelliteId: '' })
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
  const usersRead = hasPermission('users.read')
  const incidentsManage = hasPermission('incidents.manage')
  const agentsRead = hasPermission('agents.read')
  const agentsManage = hasPermission('agents.manage')
  const auditAccess = hasPermission('audit.read')
  const rbacRead = hasPermission('rbac.read')
  const rbacManage = hasPermission('rbac.manage')
  const adminAccess = admin || usersRead || auditAccess || rbacRead || rbacManage
  const connectivityAccess = agentsRead || agentsManage
  const tabs = [
    { id: 'orbital', label: t('tabs.orbital') },
    { id: 'assets', label: t('tabs.assets') },
    { id: 'signal', label: t('tabs.signal') },
    { id: 'incidents', label: t('tabs.incidents') },
    { id: 'intel', label: t('tabs.intel') },
    { id: 'decoder', label: t('tabs.decoder') },
    { id: 'connectivity', label: t('tabs.connectivity') },
    ...(adminAccess ? [{ id: 'admin', label: t('tabs.admin') }] : []),
    ...(platformAdmin ? [{ id: 'platform', label: t('tabs.platform') }] : []),
  ]

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

    async function safeFetch(fetcher, setter, fallback = []) {
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

      if (activeTab === 'orbital') {
        await safeFetch(fetchSatellites, setSatellites)
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
        const requests = []
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
  }, [onSessionExpired, activeTab, admin, auditAccess, assetsOffset, cvesOffset, cveFilters, auditOffset, incidentsOffset, incidentFilters, platformAdmin, adminAccess, usersRead, rbacRead, programsAccess, retryNonce])

  useEffect(() => {
    if (activeTab !== 'orbital') {
      setSatellitesLive(false)
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
      // Polling will retry; mutation components already surface their own errors.
    }
  }
  function handleCveFiltersChange(next) {
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
  function handleIncidentFiltersChange(next) {
    setIncidentsOffset(0)
    setIncidentFilters(next)
  }

  return (
    <div className="min-h-screen bg-transparent text-lyra-text">
      <Header
        onLogout={onLogout}
        onHome={() => onChangeTab('orbital')}
        onOpenSettings={() => onChangeTab('settings')}
        backendStatus={backendStatus}
        dbStatus={dbStatus}
        sdrStatus={sdrStatus}
        searchAssets={assets}
        searchSatellites={satellites}
        searchCves={spaceCVEs}
        onSearchNavigate={onChangeTab}
      />
      <ApiStatusBanner
        issue={apiIssue}
        onRetry={apiIssue?.retryable ? retryFailedRequests : undefined}
        onDismiss={() => setApiIssue(null)}
      />

      <main id="main-content" className="mx-auto max-w-7xl px-6 py-10 sm:px-8">
        <Tabs tabs={tabs} active={activeTab} onChange={onChangeTab} />

        {activeTab === 'orbital' && (
          <div className="grid gap-6">
            <Panel
              eyebrow={t('panels.groundTrack')}
              title={t('panels.liveOrbitalPositions')}
              uppercase
              bleed
              titleExtra={
                satellitesLive && (
                  <span className="flex items-center gap-1.5 rounded-full border border-lyra-good/30 bg-lyra-good/10 px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider text-lyra-good">
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
                  <WorldMap satellites={satellites} />
                </div>
              )}
            </Panel>
            <Panel eyebrow={t('panels.orbitalTracking')} title="Orbital altitude profile">
              {satellites === null ? <Skeleton /> : <OrbitalAltitudeChart satellites={satellites} />}
            </Panel>
            <Panel eyebrow={t('panels.orbitalTracking')} title={t('panels.trackedSatellites')} uppercase>
              {hasPermission('orbital.manage') && <NewSatelliteForm onCreated={reloadSatellites} />}
              {satellites === null ? <Skeleton /> : <SatellitesPanel satellites={satellites} canManage={hasPermission('orbital.manage')} onChanged={reloadSatellites} />}
            </Panel>
            {(hasPermission('orbital.read') || hasPermission('orbital.manage')) && (
              <Panel eyebrow={t('panels.orbitalOperations')} title={t('orbitalOperations.title')}>
                {satellites === null ? <Skeleton /> : <OrbitalOperationsPanel satellites={satellites} />}
              </Panel>
            )}
          </div>
        )}

        {activeTab === 'assets' && (
          <Panel eyebrow={t('panels.assetInventory')} title={t('panels.trackedAssets')}>
            {hasPermission('assets.manage') && programsAccess && (
              <NewAssetForm
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
                  <RfPanel readings={rfReadings} />
                </div>
              )}
            </Panel>
            <Panel eyebrow={t('panels.signalIntelligence')} title="RF station coverage">
              {rfReadings === null || agentIdentities === null ? <Skeleton /> : (
                <RFStationMapChart agents={agentIdentities} readings={rfReadings} />
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
          <Panel eyebrow={t('panels.tmtc')} title={t('panels.protocolDecoder')}>
            <Decoder />
          </Panel>
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
            <Panel eyebrow={t('platform.eyebrow')} title={t('platform.title')} uppercase>
              {organizations === null || platformUsers === null ? <Skeleton /> : <PlatformOrganizationsPanel organizations={organizations} platformUsers={platformUsers} onChanged={reloadOrganizations} onPlatformUsersChanged={reloadPlatformUsers} />}
            </Panel>
            <Panel eyebrow={t('signingKeys.eyebrow')} title={t('signingKeys.title')} uppercase>
              <SigningKeysPanel />
            </Panel>
            <Panel eyebrow={t('rateLimits.eyebrow')} title={t('rateLimits.title')} uppercase>
              <RateLimitPolicyPanel />
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
                  <RBACPanel permissions={rbacPermissions} roles={rbacRoles} users={users ?? []} onChanged={reloadRBAC} />
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
  )
}
