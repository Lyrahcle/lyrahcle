import { useEffect, useState } from 'react'
import Header from './components/Header.jsx'
import StatCard from './components/StatCard.jsx'
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

  const kinds = new Set(assets.map((a) => a.kind))

  return (
    <div className="min-h-screen bg-lyra-bg text-lyra-text">
      <Header />

      <main className="mx-auto max-w-5xl px-8 py-10">
        <div className="mb-8 grid grid-cols-3 gap-4">
          <StatCard label="Backend" value={backendStatus} />
          <StatCard label="Assets tracked" value={assets.length} />
          <StatCard label="RF readings" value={rfReadings.length} />
        </div>

        <section className="mb-8 rounded-lg border border-lyra-panel bg-lyra-panel/20 p-6">
          <h2 className="font-display mb-4 text-lg text-lyra-goldLight">Tracked assets</h2>
          {error ? (
            <p className="text-sm text-red-400">{error}</p>
          ) : (
            <AssetsTable assets={assets} />
          )}
        </section>

        <section className="rounded-lg border border-lyra-panel bg-lyra-panel/20 p-6">
          <h2 className="font-display mb-4 text-lg text-lyra-goldLight">RF spectrum</h2>
          <RfPanel readings={rfReadings} />
        </section>
      </main>
    </div>
  )
}

export default App
