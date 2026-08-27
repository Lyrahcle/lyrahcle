import { useEffect, useState } from 'react'

function App() {
  const [status, setStatus] = useState('checking backend...')

  useEffect(() => {
    fetch('/api/v1/ping')
      .then((r) => r.json())
      .then((d) => setStatus(d.message))
      .catch(() => setStatus('backend unreachable'))
  }, [])

  return (
    <main className="min-h-screen flex items-center justify-center flex-col gap-4">
      <h1 className="text-4xl font-bold tracking-tight">Lyrahcle</h1>
      <p className="text-slate-400 italic">Never Lies.</p>
      <p className="text-sm text-slate-500">{status}</p>
    </main>
  )
}

export default App
