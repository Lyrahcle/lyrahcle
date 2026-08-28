import { useEffect, useRef, useState } from 'react'
import Header from './Header.jsx'
import StatusStrip from './StatusStrip.jsx'
import Panel from './Panel.jsx'
import Tabs from './Tabs.jsx'
import Skeleton from './Skeleton.jsx'
import AssetsTable from './AssetsTable.jsx'
import RfPanel from './RfPanel.jsx'
import AnomaliesPanel from './AnomaliesPanel.jsx'
import SatellitesPanel from './SatellitesPanel.jsx'
import WorldMap from './WorldMap.jsx'
import ThreatIntelPanel from './ThreatIntelPanel.jsx'
import Decoder from './Decoder.jsx'
import NewAssetForm from './NewAssetForm.jsx'
import NewSatelliteForm from './NewSatelliteForm.jsx'
import AgentKeysPanel from './AgentKeysPanel.jsx'
import AuditLogPanel from './AuditLogPanel.jsx'
import UsersPanel from './UsersPanel.jsx'
import SettingsPanel from './SettingsPanel.jsx'
import Footer from './Footer.jsx'
import { canWrite, isAdmin } from '../lib/auth.js'
import { useI18n } from '../lib/I18nContext.jsx'
import {
  fetchAgentKeys,
  fetchAssets,
  fetchAuditLog,
  fetchPing,
  fetchRfAnomalies,
  fetchRfReadings,
  fetchSatellites,
  fetchSpaceCVEs,
  fetchUsers,
  UnauthorizedError,
} from '../lib/api.js'

const REFRESH_MS = 8000
const PAGE_LIMIT = 50

export default function Dashboard({ onLogout, onSessionExpired, activeTab, onChangeTab }) {
  const { t } = useI18n()
  const [assets, setAssets] = useState(null)
  const [rfReadings, setRfReadings] = useState(null)
  const [rfAnomalies, setRfAnomalies] = useState(null)
  const [satellites, setSatellites] = useState(null)
  const [spaceCVEs, setSpaceCVEs] = useState(null)
  const [agentKeys, setAgentKeys] = useState(null)
  const [auditEntries, setAuditEntries] = useState(null)
  const [users, setUsers] = useState(null)
  const [backendStatus, setBackendStatus] = useState('checking')
  const [dbStatus, setDbStatus] = useState('checking')

  const [assetsOffset, setAssetsOffset] = useState(0)
  const [cvesOffset, setCvesOffset] = useState(0)
  const [auditOffset, setAuditOffset] = useState(0)
  const assetsOffsetRef = useRef(0)
  const cvesOffsetRef = useRef(0)
  const auditOffsetRef = useRef(0)
  useEffect(() => {
    assetsOffsetRef.current = assetsOffset
  }, [assetsOffset])
  useEffect(() => {
    cvesOffsetRef.current = cvesOffset
  }, [cvesOffset])
  useEffect(() => {
    auditOffsetRef.current = auditOffset
  }, [auditOffset])

  const admin = isAdmin()
  const tabs = [
    { id: 'orbital', label: t('tabs.orbital') },
    { id: 'assets', label: t('tabs.assets') },
    { id: 'signal', label: t('tabs.signal') },
    { id: 'intel', label: t('tabs.intel') },
    { id: 'decoder', label: t('tabs.decoder') },
    ...(admin ? [{ id: 'admin', label: t('tabs.admin') }] : []),
    { id: 'settings', label: t('tabs.settings') },
  ]

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
      try {
        await fetchPing()
        if (!cancelled) setBackendStatus('online')
      } catch {
        if (!cancelled) setBackendStatus('offline')
      }

      try {
        const data = await fetchAssets(PAGE_LIMIT, assetsOffsetRef.current)
        if (!cancelled) {
          setAssets(data ?? { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
          setDbStatus('online')
        }
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          onSessionExpired()
          return
        }
        if (!cancelled) {
          setAssets((prev) => prev ?? { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 })
          setDbStatus('offline')
        }
      }

      await safeFetch(fetchRfReadings, setRfReadings)
      await safeFetch(fetchRfAnomalies, setRfAnomalies)
      await safeFetch(fetchSatellites, setSatellites)
      await safeFetch(
        () => fetchSpaceCVEs(PAGE_LIMIT, cvesOffsetRef.current),
        setSpaceCVEs,
        { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
      )

      if (admin) {
        await safeFetch(fetchAgentKeys, setAgentKeys)
        await safeFetch(
          () => fetchAuditLog(PAGE_LIMIT, auditOffsetRef.current),
          setAuditEntries,
          { items: [], total: 0, limit: PAGE_LIMIT, offset: 0 },
        )
        await safeFetch(fetchUsers, setUsers)
      }
    }

    poll()
    const id = setInterval(poll, REFRESH_MS)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [onSessionExpired, admin, assetsOffset, cvesOffset, auditOffset])

  const lastRf = rfReadings?.[0]
  const sdrStatus =
    lastRf && Date.now() - new Date(lastRf.captured_at).getTime() < 30000 ? 'live' : 'idle'

  function reloadAssets() {
    setAssetsOffset(0)
    fetchAssets(PAGE_LIMIT, 0).then(setAssets).catch(() => {})
  }
  function reloadSatellites() {
    fetchSatellites().then(setSatellites).catch(() => {})
  }
  function reloadAgentKeys() {
    fetchAgentKeys().then(setAgentKeys).catch(() => {})
  }
  function reloadUsers() {
    fetchUsers().then(setUsers).catch(() => {})
  }

  return (
    <div className="min-h-screen bg-transparent text-lyra-text">
      <Header onLogout={onLogout} onHome={() => onChangeTab('orbital')} onOpenSettings={() => onChangeTab('settings')} />
      <StatusStrip backendStatus={backendStatus} dbStatus={dbStatus} sdrStatus={sdrStatus} />

      <main id="main-content" className="mx-auto max-w-7xl px-6 py-10 sm:px-8">
        <Tabs tabs={tabs} active={activeTab} onChange={onChangeTab} />

        {activeTab === 'orbital' && (
          <div className="grid gap-6">
            <Panel eyebrow={t('panels.groundTrack')} title={t('panels.liveOrbitalPositions')} bleed>
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
            <Panel eyebrow={t('panels.orbitalTracking')} title={t('panels.trackedSatellites')}>
              {canWrite() && <NewSatelliteForm onCreated={reloadSatellites} />}
              {satellites === null ? <Skeleton /> : <SatellitesPanel satellites={satellites} />}
            </Panel>
          </div>
        )}

        {activeTab === 'assets' && (
          <Panel eyebrow={t('panels.assetInventory')} title={t('panels.trackedAssets')}>
            {canWrite() && <NewAssetForm onCreated={reloadAssets} />}
            {assets === null ? (
              <Skeleton />
            ) : (
              <AssetsTable
                page={assets}
                onPrev={() => setAssetsOffset((o) => Math.max(0, o - PAGE_LIMIT))}
                onNext={() => setAssetsOffset((o) => o + PAGE_LIMIT)}
              />
            )}
          </Panel>
        )}

        {activeTab === 'signal' && (
          <div className="grid gap-6">
            <Panel eyebrow={t('panels.signalIntelligence')} title={t('panels.rfSpectrum')}>
              {rfReadings === null ? <Skeleton /> : <RfPanel readings={rfReadings} />}
            </Panel>
            <Panel eyebrow={t('panels.anomalyDetection')} title={t('panels.flaggedAnomalies')}>
              {rfAnomalies === null ? <Skeleton /> : <AnomaliesPanel anomalies={rfAnomalies} />}
            </Panel>
          </div>
        )}

        {activeTab === 'intel' && (
          <Panel eyebrow={t('panels.threatIntelligence')} title={t('panels.spaceCves')}>
            {spaceCVEs === null ? (
              <Skeleton />
            ) : (
              <ThreatIntelPanel
                page={spaceCVEs}
                onPrev={() => setCvesOffset((o) => Math.max(0, o - PAGE_LIMIT))}
                onNext={() => setCvesOffset((o) => o + PAGE_LIMIT)}
              />
            )}
          </Panel>
        )}

        {activeTab === 'decoder' && (
          <Panel eyebrow={t('panels.tmtc')} title={t('panels.protocolDecoder')}>
            <Decoder />
          </Panel>
        )}

        {activeTab === 'admin' && admin && (
          <div className="grid gap-6">
            <Panel eyebrow={t('panels.accessControl')} title={t('panels.usersAndRoles')}>
              {users === null ? <Skeleton /> : <UsersPanel users={users} onChanged={reloadUsers} />}
            </Panel>
            <Panel eyebrow={t('panels.sdrAgents')} title={t('panels.agentApiKeys')}>
              {agentKeys === null ? <Skeleton /> : <AgentKeysPanel keys={agentKeys} onChanged={reloadAgentKeys} />}
            </Panel>
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
          </div>
        )}

        {activeTab === 'settings' && (
          <Panel eyebrow={t('settings.eyebrow')} title={t('settings.title')}>
            <SettingsPanel />
          </Panel>
        )}
      </main>

      <Footer />
    </div>
  )
}
