import { useI18n } from '../lib/I18nContext.jsx'

export default function Skeleton({ rows = 4 }) {
  const { t } = useI18n()
  return (
    <div role="status" aria-label={t('common.loading')} className="space-y-2.5">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} aria-hidden="true" className="h-8 motion-safe:animate-pulse rounded bg-lyra-border/40" />
      ))}
    </div>
  )
}
