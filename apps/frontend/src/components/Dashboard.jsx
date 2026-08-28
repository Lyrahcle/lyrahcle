import { useEffect, useState } from 'react'
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
import Footer from './Footer.jsx'
import { canWrite, isAdmin } from '../lib/auth.js'
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

export default function Dashboard({ onLogout, onSessionExpired, activeTab, onChangeTab }) {
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

  const admin = isAdmin()
  const tabs = [
    { id: 'orbital', label: 'Orbital' },
    { id: 'assets', label: 'Assets' },
    { id: 'signal', label: 'Signal' },
    { id: 'intel', label: 'Intel' },
    { id: 'decoder', label: 'Decoder' },
    ...(admin ? [{ id: 'admin', label: 'Admin' }] : []),
  ]

  useEffect(() => {
    let cancelled = false

    async function safeFetch(fetcher, setter) {
      try {
        const data = await fetcher()
        if (!cancelled) setter(data ?? [])
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          onSessionExpired()
          return
        }
        if (!cancelled) setter((prev) => prev ?? [])
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
        const data = await fetchAssets()
        if (!cancelled) {
          setAssets(data ?? [])
          setDbStatus('online')
        }
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          onSessionExpired()
          return
        }
        if (!cancelled) {
          setAssets((prev) => prev ?? [])
          setDbStatus('offline')
        }
      }

      await safeFetch(fetchRfReadings, setRfReadings)
      await safeFetch(fetchRfAnomalies, setRfAnomalies)
      await safeFetch(fetchSatellites, setSatellites)
      await safeFetch(fetchSpaceCVEs, setSpaceCVEs)

      if (admin) {
        await safeFetch(fetchAgentKeys, setAgentKeys)
        await safeFetch(fetchAuditLog, setAuditEntries)
        await safeFetch(fetchUsers, setUsers)
      }
    }

    poll()
    const id = setInterval(poll, REFRESH_MS)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [onSessionExpired, admin])

  const lastRf = rfReadings?.[0]
  const sdrStatus =
    lastRf && Date.now() - new Date(lastRf.captured_at).getTime() < 30000 ? 'live' : 'idle'

  function reloadAssets() {
    fetchAssets().then(setAssets).catch(() => {})
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
      <Header onLogout={onLogout} onHome={() => onChangeTab('orbital')} />
      <StatusStrip backendStatus={backendStatus} dbStatus={dbStatus} sdrStatus={sdrStatus} />

      <main id="main-content" className="mx-auto max-w-7xl px-6 py-10 sm:px-8">
        <Tabs tabs={tabs} active={activeTab} onChange={onChangeTab} />

        {activeTab === 'orbital' && (
          <div className="grid gap-6">
            <Panel eyebrow="Ground Track" title="Live orbital positions" bleed>
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
            <Panel eyebrow="Orbital Tracking" title="Tracked satellites">
              {canWrite() && <NewSatelliteForm onCreated={reloadSatellites} />}
              {satellites === null ? <Skeleton /> : <SatellitesPanel satellites={satellites} />}
            </Panel>
          </div>
        )}

        {activeTab === 'assets' && (
          <Panel eyebrow="Asset Inventory" title="Tracked assets">
            {canWrite() && <NewAssetForm onCreated={reloadAssets} />}
            {assets === null ? <Skeleton /> : <AssetsTable assets={assets} />}
          </Panel>
        )}

        {activeTab === 'signal' && (
          <div className="grid gap-6">
            <Panel eyebrow="Signal Intelligence" title="RF spectrum">
              {rfReadings === null ? <Skeleton /> : <RfPanel readings={rfReadings} />}
            </Panel>
            <Panel eyebrow="Anomaly Detection" title="Flagged RF anomalies">
              {rfAnomalies === null ? <Skeleton /> : <AnomaliesPanel anomalies={rfAnomalies} />}
            </Panel>
          </div>
        )}

        {activeTab === 'intel' && (
          <Panel eyebrow="Threat Intelligence" title="Space-related CVEs">
            {spaceCVEs === null ? <Skeleton /> : <ThreatIntelPanel cves={spaceCVEs} />}
          </Panel>
        )}

        {activeTab === 'decoder' && (
          <Panel eyebrow="TM/TC" title="Protocol decoder">
            <Decoder />
          </Panel>
        )}

        {activeTab === 'admin' && admin && (
          <div className="grid gap-6">
            <Panel eyebrow="Access Control" title="Users & roles">
              {users === null ? <Skeleton /> : <UsersPanel users={users} onChanged={reloadUsers} />}
            </Panel>
            <Panel eyebrow="SDR Agents" title="Agent API keys">
              {agentKeys === null ? <Skeleton /> : <AgentKeysPanel keys={agentKeys} onChanged={reloadAgentKeys} />}
            </Panel>
            <Panel eyebrow="Security" title="Audit log">
              {auditEntries === null ? <Skeleton /> : <AuditLogPanel entries={auditEntries} />}
            </Panel>
          </div>
        )}
      </main>

      <Footer />
    </div>
  )
}
