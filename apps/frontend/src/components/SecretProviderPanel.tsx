import { useEffect, useState } from 'react'
import { fetchPlatformSecretStatus } from '../lib/api'
import { useI18n } from '../lib/I18nContext'

interface SecretProviderState {
  provider: string
  healthy: boolean
  external: boolean
  location?: string
  managed_keys?: string[]
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

export default function SecretProviderPanel() {
  const { t } = useI18n()
  const [state, setState] = useState<SecretProviderState | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    fetchPlatformSecretStatus()
      .then((result) => { if (!cancelled) setState(result as SecretProviderState) })
      .catch((error: unknown) => { if (!cancelled) setError(errorMessage(error)) })
    return () => { cancelled = true }
  }, [])

  if (!state && !error) return <p className="text-sm text-lyra-muted">{t('secretProvider.loading')}</p>
  if (error) return <p role="alert" className="text-sm text-lyra-bad">{error}</p>
  if (!state) return null

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <span className="rounded border border-lyra-border px-2 py-1 font-mono text-xs text-lyra-goldBright">{state.provider}</span>
        <span className={`rounded border px-2 py-1 font-mono text-xs ${state.healthy ? 'border-lyra-ok/50 text-lyra-ok' : 'border-lyra-bad/50 text-lyra-bad'}`}>
          {state.healthy ? t('secretProvider.healthy') : t('secretProvider.unhealthy')}
        </span>
        <span className="font-mono text-xs text-lyra-muted">{state.external ? t('secretProvider.externalCustody') : t('secretProvider.localCustody')}</span>
      </div>
      {state.location && <p className="break-all font-mono text-xs text-lyra-muted">{t('secretProvider.location', { location: state.location })}</p>}
      <div>
        <p className="mb-2 text-xs tracking-wider text-lyra-mutedDim">{t('secretProvider.managedRefs')}</p>
        <div className="flex flex-wrap gap-2">
          {(state.managed_keys ?? []).map((key) => <span key={key} className="rounded bg-lyra-bgSoft px-2 py-1 font-mono text-[11px] text-lyra-muted">{key}</span>)}
        </div>
      </div>
      <p className="text-[11px] text-lyra-mutedDim">{t('secretProvider.disclaimer')}</p>
    </div>
  )
}
