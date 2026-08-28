import { useEffect, useState } from 'react'
import Header from './Header.jsx'
import StatusStrip from './StatusStrip.jsx'
import Panel from './Panel.jsx'
import Tabs from './Tabs.jsx'
import Skeleton from './Skeleton.jsx'
import AssetsTable from './AssetsTable.jsx'
import RfPanel from './RfPanel.jsx'
import SatellitesPanel from './SatellitesPanel.jsx'
import WorldMap from './WorldMap.jsx'
import ThreatIntelPanel from './ThreatIntelPanel.jsx'
import Decoder from './Decoder.jsx'
import Footer from './Footer.jsx'
import {
  fetchAssets,
  fetchPing,
  fetchRfReadings,
  fetchSatellites,
  fetchSpaceCVEs,
  UnauthorizedError,
} from '../lib/api.js'

const REFRESH_MS = 8000

const TABS = [
  { id: 'orbital', label: 'Orbital' },
  { id: 'assets', label: 'Assets' },
  { id: 'signal', label: 'Signal' },
  { id: 'intel', label: 'Intel' },
  { id: 'decoder', label: 'Decoder' },
]

export default function Dashboard({ onLogout, onSessionExpired, activeTab, onChangeTab }) {
  const [assets, setAssets] = useState(null)
  const [rfReadings, setRfReadings] = useState(null)
  const [satellites, setSatellites] = useState(null)
  const [spaceCVEs, setSpaceCVEs] = useState(null)
  const [backendStatus, setBackendStatus] = useState('checking')
  const [dbStatus, setDbStatus] = useState('checking')

  useEffect(() => {
    let cancelled = false

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

      try {
        const data = await fetchRfReadings()
        if (!cancelled) setRfReadings(data ?? [])
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          onSessionExpired()
          return
        }
        if (!cancelled) setRfReadings((prev) => prev ?? [])
      }

      try {
        const data = await fetchSatellites()
        if (!cancelled) setSatellites(data ?? [])
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          onSessionExpired()
          return
        }
        if (!cancelled) setSatellites((prev) => prev ?? [])
      }

      try {
        const data = await fetchSpaceCVEs()
        if (!cancelled) setSpaceCVEs(data ?? [])
      } catch (err) {
        if (err instanceof UnauthorizedError) {
          onSessionExpired()
          return
        }
        if (!cancelled) setSpaceCVEs((prev) => prev ?? [])
      }
    }

    poll()
    const id = setInterval(poll, REFRESH_MS)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [onSessionExpired])

  const lastRf = rfReadings?.[0]
  const sdrStatus =
    lastRf && Date.now() - new Date(lastRf.captured_at).getTime() < 30000 ? 'live' : 'idle'

  return (
    <div className="min-h-screen bg-transparent text-lyra-text">
      <Header onLogout={onLogout} onHome={() => onChangeTab('orbital')} />
      <StatusStrip backendStatus={backendStatus} dbStatus={dbStatus} sdrStatus={sdrStatus} />

      <main id="main-content" className="mx-auto max-w-7xl px-6 py-10 sm:px-8">
        <Tabs tabs={TABS} active={activeTab} onChange={onChangeTab} />

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
              {satellites === null ? <Skeleton /> : <SatellitesPanel satellites={satellites} />}
            </Panel>
          </div>
        )}

        {activeTab === 'assets' && (
          <Panel eyebrow="Asset Inventory" title="Tracked assets">
            {assets === null ? <Skeleton /> : <AssetsTable assets={assets} />}
          </Panel>
        )}

        {activeTab === 'signal' && (
          <Panel eyebrow="Signal Intelligence" title="RF spectrum">
            {rfReadings === null ? <Skeleton /> : <RfPanel readings={rfReadings} />}
          </Panel>
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
      </main>

      <Footer />
    </div>
  )
}
