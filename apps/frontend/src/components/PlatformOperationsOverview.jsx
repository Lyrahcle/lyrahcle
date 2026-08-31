import { useEffect, useMemo, useState } from 'react'
import { Activity, Building2, KeyRound, ShieldCheck, Users } from 'lucide-react'
import { fetchAgentConfigSigningKeys, fetchPlatformReadiness, fetchPlatformSecretStatus } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'

function tone(status) {
  if (status === 'healthy') return 'border-lyra-ok/40 text-lyra-ok'
  if (status === 'degraded') return 'border-lyra-warn/40 text-lyra-warn'
  if (status === 'unavailable') return 'border-lyra-bad/40 text-lyra-bad'
  return 'border-lyra-border text-lyra-muted'
}

function Metric({ icon: Icon, label, value, status = 'neutral' }) {
  return (
    <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-[10px] uppercase tracking-[0.15em] text-lyra-muted">{label}</p>
        <Icon aria-hidden="true" className="h-4 w-4 text-lyra-mutedDim" />
      </div>
      <p className={`mt-2 inline-flex rounded border px-2 py-1 font-mono text-sm uppercase ${tone(status)}`}>{value}</p>
    </div>
  )
}

export default function PlatformOperationsOverview({ organizations = [], platformUsers = [] }) {
  const { t } = useI18n()
  const [readiness, setReadiness] = useState(null)
  const [secrets, setSecrets] = useState(null)
  const [signing, setSigning] = useState(null)
  const [errors, setErrors] = useState([])

  useEffect(() => {
    let cancelled = false
    Promise.allSettled([
      fetchPlatformReadiness(),
      fetchPlatformSecretStatus(),
      fetchAgentConfigSigningKeys(),
    ]).then((results) => {
      if (cancelled) return
      const nextErrors = []
      if (results[0].status === 'fulfilled') setReadiness(results[0].value)
      else nextErrors.push('readiness')
      if (results[1].status === 'fulfilled') setSecrets(results[1].value)
      else nextErrors.push('secrets')
      if (results[2].status === 'fulfilled') setSigning(results[2].value)
      else nextErrors.push('signing')
      setErrors(nextErrors)
    })
    return () => { cancelled = true }
  }, [])

  const platformAdmins = useMemo(
    () => platformUsers.filter((user) => Boolean(user.platform_admin)).length,
    [platformUsers],
  )
  const signingHealthy = Boolean(signing?.managed && signing?.health && !signing.health.rotation_due && !(signing.health.warnings ?? []).length)
  const readinessHealthy = readiness?.status === 'ready'
  const overall = errors.length > 0 || readiness === null || secrets === null || signing === null
    ? 'degraded'
    : readinessHealthy && secrets?.healthy && signingHealthy ? 'healthy' : 'degraded'

  return (
    <section aria-labelledby="platform-operations-heading" className="grid gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-[10px] uppercase tracking-[0.18em] text-lyra-mutedDim">{t('platformOperations.eyebrow')}</p>
          <h2 id="platform-operations-heading" className="mt-1 font-display text-xl font-medium text-lyra-text">{t('platformOperations.title')}</h2>
          <p className="mt-1 max-w-2xl text-sm text-lyra-muted">{t('platformOperations.description')}</p>
        </div>
        <span className={`self-start rounded border px-2.5 py-1 font-mono text-xs uppercase sm:self-auto ${tone(overall)}`}>
          {t(`platformOperations.${overall}`)}
        </span>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <Metric icon={Activity} label={t('platformOperations.core')} value={readiness === null ? t('platformOperations.checking') : readinessHealthy ? t('platformOperations.ready') : t('platformOperations.notReady')} status={readiness === null ? 'neutral' : readinessHealthy ? 'healthy' : 'unavailable'} />
        <Metric icon={ShieldCheck} label={t('platformOperations.secrets')} value={secrets === null ? t('platformOperations.checking') : secrets.healthy ? t('platformOperations.healthy') : t('platformOperations.unhealthy')} status={secrets === null ? 'neutral' : secrets.healthy ? 'healthy' : 'unavailable'} />
        <Metric icon={KeyRound} label={t('platformOperations.signing')} value={signing === null ? t('platformOperations.checking') : signingHealthy ? t('platformOperations.healthy') : t('platformOperations.attention')} status={signing === null ? 'neutral' : signingHealthy ? 'healthy' : 'degraded'} />
        <Metric icon={Building2} label={t('platformOperations.organizations')} value={String(organizations.length)} />
        <Metric icon={Users} label={t('platformOperations.platformAdmins')} value={String(platformAdmins)} />
      </div>

      {errors.length > 0 && (
        <p role="alert" className="rounded border border-lyra-warn/30 bg-lyra-warn/5 px-3 py-2 text-xs text-lyra-warn">
          {t('platformOperations.partial')}: {errors.join(', ')}
        </p>
      )}
      <p className="text-[11px] text-lyra-mutedDim">{t('platformOperations.readinessHint')}</p>
    </section>
  )
}
