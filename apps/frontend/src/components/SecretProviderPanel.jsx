import { useEffect, useState } from 'react'
import { fetchPlatformSecretStatus } from '../lib/api.js'

export default function SecretProviderPanel() {
  const [state, setState] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    let cancelled = false
    fetchPlatformSecretStatus()
      .then((result) => { if (!cancelled) setState(result) })
      .catch((err) => { if (!cancelled) setError(err.message) })
    return () => { cancelled = true }
  }, [])

  if (!state && !error) return <p className="text-sm text-lyra-muted">Loading secret provider status…</p>
  if (error) return <p role="alert" className="text-sm text-lyra-bad">{error}</p>

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <span className="rounded border border-lyra-border px-2 py-1 font-mono text-xs uppercase text-lyra-goldBright">{state.provider}</span>
        <span className={`rounded border px-2 py-1 font-mono text-xs uppercase ${state.healthy ? 'border-lyra-ok/50 text-lyra-ok' : 'border-lyra-bad/50 text-lyra-bad'}`}>
          {state.healthy ? 'Healthy' : 'Unhealthy'}
        </span>
        <span className="font-mono text-xs text-lyra-muted">{state.external ? 'External custody' : 'Local custody'}</span>
      </div>
      {state.location && <p className="break-all font-mono text-xs text-lyra-muted">Location: {state.location}</p>}
      <div>
        <p className="mb-2 text-xs uppercase tracking-wider text-lyra-mutedDim">Managed secret references</p>
        <div className="flex flex-wrap gap-2">
          {(state.managed_keys ?? []).map((key) => <span key={key} className="rounded bg-lyra-bgSoft px-2 py-1 font-mono text-[11px] text-lyra-muted">{key}</span>)}
        </div>
      </div>
      <p className="text-[11px] text-lyra-mutedDim">Secret values are never returned by this endpoint or rendered in the browser. Provider writes and rotation remain in the external custody system.</p>
    </div>
  )
}
