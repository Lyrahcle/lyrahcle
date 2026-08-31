import { Clock3, Orbit, RadioTower, TriangleAlert } from 'lucide-react'
import StatCard from './StatCard.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

const DAY_MS = 24 * 60 * 60 * 1000

function hasPosition(satellite) {
  const position = satellite?.position
  return Number.isFinite(Number(position?.latitude_deg)) &&
    Number.isFinite(Number(position?.longitude_deg)) &&
    Number.isFinite(Number(position?.altitude_km))
}

function ageMs(value, now) {
  const time = new Date(value ?? 0).getTime()
  return Number.isFinite(time) ? Math.max(0, now - time) : Number.POSITIVE_INFINITY
}

function scrollTo(id) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

export default function OrbitalDashboardOverview({ satellites = [], live = false, now = Date.now() }) {
  const { t } = useI18n()
  const positioned = satellites.filter(hasPosition)
  const stale = satellites.filter((satellite) => ageMs(satellite.updated_at, now) > DAY_MS)
  const missingPosition = satellites.filter((satellite) => !hasPosition(satellite))
  const recent = [...satellites]
    .sort((a, b) => new Date(b.updated_at ?? 0).getTime() - new Date(a.updated_at ?? 0).getTime())
    .slice(0, 4)

  const attention = [
    ...stale.map((satellite) => ({ id: `stale-${satellite.id}`, name: satellite.name, reason: t('orbitalDashboard.staleTle') })),
    ...missingPosition
      .filter((satellite) => !stale.some((item) => item.id === satellite.id))
      .map((satellite) => ({ id: `position-${satellite.id}`, name: satellite.name, reason: t('orbitalDashboard.noPosition') })),
  ].slice(0, 5)

  return (
    <section aria-labelledby="orbital-dashboard-heading" className="grid gap-5">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-[10px] uppercase tracking-[0.18em] text-lyra-mutedDim">{t('orbitalDashboard.eyebrow')}</p>
          <h1 id="orbital-dashboard-heading" className="mt-1 font-display text-2xl font-medium text-lyra-text">{t('orbitalDashboard.title')}</h1>
          <p className="mt-1 max-w-2xl text-sm text-lyra-muted">{t('orbitalDashboard.subtitle')}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button type="button" onClick={() => scrollTo('orbital-ground-track')} className="rounded border border-lyra-border bg-lyra-panel px-3 py-2 text-xs text-lyra-text hover:border-lyra-gold/40 hover:text-lyra-goldBright">{t('orbitalDashboard.openMap')}</button>
          <button type="button" onClick={() => scrollTo('orbital-operations')} className="rounded border border-lyra-gold/30 bg-lyra-gold/10 px-3 py-2 text-xs text-lyra-goldBright hover:bg-lyra-gold/15">{t('orbitalDashboard.openOperations')}</button>
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={Orbit} label={t('orbitalDashboard.tracked')} value={satellites.length} />
        <StatCard icon={RadioTower} label={t('orbitalDashboard.positions')} value={`${positioned.length}/${satellites.length}`} accent={positioned.length === satellites.length ? 'good' : 'neutral'} />
        <StatCard icon={Clock3} label={t('orbitalDashboard.freshTle')} value={`${Math.max(0, satellites.length - stale.length)}/${satellites.length}`} accent={stale.length ? 'neutral' : 'good'} />
        <StatCard icon={RadioTower} label={t('orbitalDashboard.stream')} value={live ? t('orbitalDashboard.live') : t('orbitalDashboard.polling')} accent={live ? 'good' : 'neutral'} />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-5">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[10px] uppercase tracking-[0.15em] text-lyra-mutedDim">{t('orbitalDashboard.attentionEyebrow')}</p>
              <h2 className="mt-1 font-display text-base font-medium text-lyra-text">{t('orbitalDashboard.attention')}</h2>
            </div>
            <TriangleAlert className={`h-4 w-4 ${attention.length ? 'text-lyra-warn' : 'text-lyra-good'}`} aria-hidden="true" />
          </div>
          {attention.length === 0 ? (
            <p className="mt-4 text-sm text-lyra-muted">{t('orbitalDashboard.allHealthy')}</p>
          ) : (
            <ul className="mt-4 grid gap-2">
              {attention.map((item) => (
                <li key={item.id} className="flex items-center justify-between gap-3 rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2">
                  <span className="truncate text-sm text-lyra-text">{item.name}</span>
                  <span className="shrink-0 text-[11px] text-lyra-warn">{item.reason}</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-5">
          <p className="text-[10px] uppercase tracking-[0.15em] text-lyra-mutedDim">{t('orbitalDashboard.recentEyebrow')}</p>
          <h2 className="mt-1 font-display text-base font-medium text-lyra-text">{t('orbitalDashboard.recent')}</h2>
          {recent.length === 0 ? (
            <p className="mt-4 text-sm text-lyra-muted">{t('orbitalDashboard.empty')}</p>
          ) : (
            <ul className="mt-4 divide-y divide-lyra-borderSoft">
              {recent.map((satellite) => (
                <li key={satellite.id} className="flex items-center justify-between gap-3 py-2.5 first:pt-0 last:pb-0">
                  <div className="min-w-0">
                    <p className="truncate text-sm text-lyra-text">{satellite.name}</p>
                    <p className="font-mono text-[10px] text-lyra-mutedDim">NORAD {satellite.norad_id ?? '—'}</p>
                  </div>
                  <time className="shrink-0 text-[11px] text-lyra-muted" dateTime={satellite.updated_at ?? undefined}>
                    {satellite.updated_at ? new Date(satellite.updated_at).toLocaleString() : '—'}
                  </time>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </section>
  )
}
