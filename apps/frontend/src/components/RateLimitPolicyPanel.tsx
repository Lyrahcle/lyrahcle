import { useEffect, useState } from 'react'
import { ApiError, fetchPlatformRateLimits } from '../lib/api'
import { useI18n } from '../lib/I18nContext'
import StatusState from './StatusState'

interface BudgetCardProps {
  label: string
  tenant?: number | null
  actor?: number | null
  suffix: string
}

interface RateLimitState {
  redis_distributed: boolean
  tenant_read_per_minute?: number
  actor_read_per_minute?: number
  tenant_write_per_minute?: number
  actor_write_per_minute?: number
  tenant_agent_per_minute?: number
  agent_per_minute?: number
  platform_ceiling_per_minute?: number
  human_and_machine_budgets_separate?: boolean
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function BudgetCard({ label, tenant, actor, suffix }: BudgetCardProps) {
  return (
    <div className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/20 p-3">
      <p className="font-mono text-[10px] tracking-wider text-lyra-mutedDim">{label}</p>
      <div className="mt-2 grid grid-cols-2 gap-3 text-sm">
        <div>
          <p className="text-xs text-lyra-muted">Tenant</p>
          <p className="font-mono text-lg text-lyra-goldBright">{tenant ?? '—'}<span className="ml-1 text-[10px] text-lyra-muted">{suffix}</span></p>
        </div>
        <div>
          <p className="text-xs text-lyra-muted">Actor</p>
          <p className="font-mono text-lg text-lyra-text">{actor ?? '—'}<span className="ml-1 text-[10px] text-lyra-muted">{suffix}</span></p>
        </div>
      </div>
    </div>
  )
}

export default function RateLimitPolicyPanel() {
  const { t } = useI18n()
  const [state, setState] = useState<RateLimitState | null>(null)
  const [error, setError] = useState<unknown>(null)
  const [reload, setReload] = useState(0)

  useEffect(() => {
    let cancelled = false
    fetchPlatformRateLimits()
      .then((result) => { if (!cancelled) setState(result as RateLimitState) })
      .catch((error: unknown) => { if (!cancelled) setError(error) })
    return () => { cancelled = true }
  }, [reload])

  if (!state && !error) return <StatusState kind="loading" title={t('ux.states.loadingTitle')} message={t('ux.states.loadingHint')} compact />
  if (error) {
    const offline = error instanceof ApiError && error.kind === 'network'
    return (
      <StatusState
        kind={offline ? 'offline' : 'error'}
        title={offline ? t('ux.states.offlineTitle') : t('ux.states.errorTitle')}
        message={errorMessage(error)}
        actionLabel={t('ux.states.retry')}
        onAction={() => { setState(null); setError(null); setReload((value) => value + 1) }}
        compact
      />
    )
  }
  if (!state) return null

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <p className="max-w-3xl text-sm text-lyra-muted">{t('rateLimits.hint')}</p>
        <span className={`rounded border px-2 py-1 font-mono text-[10px] ${state.redis_distributed ? 'border-lyra-ok/50 text-lyra-ok' : 'border-lyra-warn/50 text-lyra-warn'}`}>
          {state.redis_distributed ? t('rateLimits.distributed') : t('rateLimits.localFallback')}
        </span>
      </div>

      <div className="grid gap-3 lg:grid-cols-3">
        <BudgetCard label={t('rateLimits.reads')} tenant={state.tenant_read_per_minute} actor={state.actor_read_per_minute} suffix="/min" />
        <BudgetCard label={t('rateLimits.writes')} tenant={state.tenant_write_per_minute} actor={state.actor_write_per_minute} suffix="/min" />
        <BudgetCard label={t('rateLimits.agents')} tenant={state.tenant_agent_per_minute} actor={state.agent_per_minute} suffix="/min" />
      </div>

      <div className="grid gap-2 rounded border border-lyra-borderSoft p-3 text-xs text-lyra-muted md:grid-cols-2">
        <p>{t('rateLimits.platformCeiling')}: <span className="font-mono text-lyra-text">{state.platform_ceiling_per_minute}/min</span></p>
        <p>{t('rateLimits.separateNamespaces')}: <span className="font-mono text-lyra-ok">{state.human_and_machine_budgets_separate ? t('rateLimits.yes') : t('rateLimits.no')}</span></p>
      </div>

      <p className="text-[11px] text-lyra-mutedDim">{t('rateLimits.configHint')}</p>
    </div>
  )
}
