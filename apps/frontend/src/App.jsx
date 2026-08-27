import { useEffect, useState } from 'react'
import Header from './components/Header.jsx'
import Hero from './components/Hero.jsx'
import Panel from './components/Panel.jsx'
import Tabs from './components/Tabs.jsx'
import Skeleton from './components/Skeleton.jsx'
import AssetsTable from './components/AssetsTable.jsx'
import RfPanel from './components/RfPanel.jsx'
import SatellitesPanel from './components/SatellitesPanel.jsx'
import WorldMap from './components/WorldMap.jsx'
import { fetchAssets, fetchPing, fetchRfReadings, fetchSatellites } from './lib/api.js'

const REFRESH_MS = 8000

const TABS = [
  { id: 'orbital', label: 'Orbital' },
  { id: 'assets', label: 'Assets' },
  { id: 'signal', label: 'Signal' },
]

function App() {
  const [assets, setAssets] = useState(null)
  const [rfReadings, setRfReadings] = useState(null)
  const [satellites, setSatellites] = useState(null)
  const [backendStatus, setBackendStatus] = useState('checking')
  const [dbStatus, setDbStatus] = useState('checking')
  const [activeTab, setActiveTab] = useState('orbital')

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
      } catch {
        if (!cancelled) {
          setAssets((prev) => prev ?? [])
          setDbStatus('offline')
        }
      }

      try {
        const data = await fetchRfReadings()
        if (!cancelled) setRfReadings(data ?? [])
      } catch {
        if (!cancelled) setRfReadings((prev) => prev ?? [])
      }

      try {
        const data = await fetchSatellites()
        if (!cancelled) setSatellites(data ?? [])
      } catch {
        if (!cancelled) setSatellites((prev) => prev ?? [])
      }
    }

    poll()
    const id = setInterval(poll, REFRESH_MS)
    return () => {
      cancelled = true
      clearInterval(id)
    }
  }, [])

  const lastRf = rfReadings?.[0]
  const sdrStatus =
    lastRf && Date.now() - new Date(lastRf.captured_at).getTime() < 30000 ? 'live' : 'idle'

  return (
    <div className="min-h-screen bg-transparent text-lyra-text">
      <Header />
      <Hero backendStatus={backendStatus} dbStatus={dbStatus} sdrStatus={sdrStatus} />

      <main className="mx-auto max-w-6xl px-8 py-10">
        <Tabs tabs={TABS} active={activeTab} onChange={setActiveTab} />

        {activeTab === 'orbital' && (
          <div className="grid gap-6">
            <Panel eyebrow="Ground Track" title="Live orbital positions">
              {satellites === null ? (
                <Skeleton rows={1} />
              ) : (
                <WorldMap satellites={satellites} />
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
      </main>

      <footer className="mx-auto max-w-6xl px-8 py-8 text-center">
        <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-mutedDim">
          Lyrahcle · Never Lies
        </p>
      </footer>
    </div>
  )
}

export default App
