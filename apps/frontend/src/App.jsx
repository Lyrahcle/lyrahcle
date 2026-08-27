import { useEffect, useState } from 'react'
import { Database, Radio, Server } from 'lucide-react'
import Header from './components/Header.jsx'
import StatCard from './components/StatCard.jsx'
import Panel from './components/Panel.jsx'
import AssetsTable from './components/AssetsTable.jsx'
import RfPanel from './components/RfPanel.jsx'
import { fetchAssets, fetchPing, fetchRfReadings } from './lib/api.js'

function App() {
  const [assets, setAssets] = useState([])
  const [rfReadings, setRfReadings] = useState([])
  const [backendStatus, setBackendStatus] = useState('checking')
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchPing()
      .then(() => setBackendStatus('online'))
      .catch(() => setBackendStatus('offline'))

    fetchAssets()
      .then((data) => setAssets(data ?? []))
      .catch((e) => setError(e.message))

    fetchRfReadings()
      .then((data) => setRfReadings(data ?? []))
      .catch(() => {})
  }, [])

  return (
    <div className="min-h-screen bg-transparent text-lyra-text">
      <Header backendStatus={backendStatus} />

      <main className="mx-auto max-w-6xl px-8 py-10">
        <div className="mb-8 grid grid-cols-3 gap-4">
          <StatCard icon={Server} label="Backend status" value={backendStatus} accent="good" />
          <StatCard icon={Database} label="Assets tracked" value={assets.length} accent="gold" />
          <StatCard icon={Radio} label="RF readings" value={rfReadings.length} accent="neutral" />
        </div>

        <div className="grid gap-6">
          <Panel eyebrow="Asset Inventory" title="Tracked assets">
            {error ? (
              <p className="text-sm text-lyra-bad">{error}</p>
            ) : (
              <AssetsTable assets={assets} />
            )}
          </Panel>

          <Panel eyebrow="Signal Intelligence" title="RF spectrum">
            <RfPanel readings={rfReadings} />
          </Panel>
        </div>
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
