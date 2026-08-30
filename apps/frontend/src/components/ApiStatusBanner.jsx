import { useEffect, useState } from 'react'
import { useI18n } from '../lib/I18nContext.jsx'

function copyForKind(t, kind) {
  const known = ['forbidden', 'not_found', 'conflict', 'rate_limited', 'server', 'network']
  const key = known.includes(kind) ? kind : 'server'
  return {
    title: t(`httpErrors.${key}.title`),
    message: t(`httpErrors.${key}.message`),
  }
}

export default function ApiStatusBanner({ issue, onRetry, onDismiss }) {
  const { t } = useI18n()
  const [remaining, setRemaining] = useState(0)

  useEffect(() => {
    const initial = Math.max(0, Number(issue?.retryAfterSeconds) || 0)
    setRemaining(initial)
    if (!initial) return undefined
    const deadline = Date.now() + initial * 1000
    const timer = setInterval(() => {
      const next = Math.max(0, Math.ceil((deadline - Date.now()) / 1000))
      setRemaining(next)
      if (next <= 0) clearInterval(timer)
    }, 250)
    return () => clearInterval(timer)
  }, [issue])

  if (!issue) return null
  const copy = copyForKind(t, issue.kind)
  const canRetry = Boolean(issue.retryable && onRetry)
  const retryDisabled = issue.kind === 'rate_limited' && remaining > 0

  return (
    <section role="alert" aria-live="polite" className="border-b border-lyra-warn/30 bg-lyra-warn/10">
      <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-3 px-6 py-3 text-sm sm:px-8">
        <div className="min-w-0">
          <p className="font-medium text-lyra-text">{copy.title}</p>
          <p className="mt-0.5 text-xs text-lyra-muted">
            {copy.message}
            {issue.requestId ? ` ${t('httpErrors.requestId')}: ${issue.requestId}` : ''}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {canRetry && (
            <button
              type="button"
              onClick={onRetry}
              disabled={retryDisabled}
              className="rounded border border-lyra-gold/40 px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-lyra-goldBright disabled:cursor-not-allowed disabled:opacity-50"
            >
              {retryDisabled ? t('httpErrors.retryIn', { seconds: remaining }) : t('httpErrors.retry')}
            </button>
          )}
          <button
            type="button"
            onClick={onDismiss}
            className="rounded border border-lyra-border px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-lyra-muted"
          >
            {t('httpErrors.dismiss')}
          </button>
        </div>
      </div>
    </section>
  )
}
