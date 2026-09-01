import { useEffect, useState } from 'react'
import { fetchPlatformRateLimits } from '../lib/api'
import { useI18n } from '../lib/I18nContext'

function BudgetCard({ label, tenant, actor, suffix }) {
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
  const [state, setState] = useState(null)
  const [error, setError] = useState(null)

  useEffect(() => {
    let cancelled = false
    fetchPlatformRateLimits()
      .then((result) => { if (!cancelled) setState(result) })
      .catch((err) => { if (!cancelled) setError(err.message) })
    return () => { cancelled = true }
  }, [])

  if (!state && !error) return <p className="text-sm text-lyra-muted">{t('common.loading')}</p>
  if (error) return <p role="alert" className="text-sm text-lyra-bad">{error}</p>

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
