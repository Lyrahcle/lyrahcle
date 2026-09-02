import { useMemo } from 'react'
import { calculateMissionRisk, type RiskDomain, type RiskFactor } from '../lib/risk'
import { useI18n } from '../lib/I18nContext'

type Entity = Record<string, unknown>

type Access = {
  incidents?: boolean
  rf?: boolean
  agents?: boolean
  alerts?: boolean
}

const DOMAIN_ORDER: RiskDomain[] = ['security', 'rf', 'orbital', 'exposure', 'ground', 'platform']

export function buildMissionSecurityScore({
  satellites = [],
  incidents = [],
  anomalies = [],
  agents = [],
  alerts = [],
  backendStatus = 'checking',
  access = {},
  now = Date.now(),
}: {
  satellites?: Entity[]
  incidents?: Entity[]
  anomalies?: Entity[]
  agents?: Entity[]
  alerts?: Entity[]
  backendStatus?: string
  access?: Access
  now?: number
}) {
  const risk = calculateMissionRisk({
    satellites,
    incidents: access.incidents ? incidents : [],
    anomalies: access.rf ? anomalies : [],
    agents: access.agents ? agents : [],
    alerts: access.alerts ? alerts : [],
    backendStatus,
    now,
  })
  const availableSources = 2 + [access.incidents, access.rf, access.agents, access.alerts].filter(Boolean).length
  return {
    ...risk,
    availableSources,
    totalSources: 6,
    domains: DOMAIN_ORDER.map((domain) => ({ domain, score: risk.domains[domain] || 0 })),
    factors: [...risk.factors].sort((a, b) => b.points - a.points),
  }
}

function levelClass(level: string) {
  if (level === 'critical') return 'border-lyra-bad/40 bg-lyra-bad/10 text-lyra-bad'
  if (level === 'elevated') return 'border-lyra-warn/40 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
}

function FactorRow({ factor, label }: { factor: RiskFactor; label: string }) {
  return (
    <li className="flex items-center justify-between gap-3 rounded border border-lyra-borderSoft bg-lyra-bg/20 px-3 py-2">
      <span className="min-w-0 truncate text-xs text-lyra-muted">{label}</span>
      <span className="shrink-0 font-mono text-xs text-lyra-text">+{factor.points}</span>
    </li>
  )
}

export default function MissionSecurityScore({
  satellites = [],
  incidents = [],
  anomalies = [],
  agents = [],
  alerts = [],
  backendStatus = 'checking',
  access = {},
  now,
}: {
  satellites?: Entity[]
  incidents?: Entity[]
  anomalies?: Entity[]
  agents?: Entity[]
  alerts?: Entity[]
  backendStatus?: string
  access?: Access
  now?: number
}) {
  const { t } = useI18n()
  const result = useMemo(() => buildMissionSecurityScore({
    satellites,
    incidents,
    anomalies,
    agents,
    alerts,
    backendStatus,
    access,
    now,
  }), [satellites, incidents, anomalies, agents, alerts, backendStatus, access, now])

  return (
    <section aria-labelledby="mission-security-score-title" className="rounded-xl border border-lyra-border bg-lyra-panel/45 p-5 sm:p-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <p className="text-[10px] tracking-[0.18em] text-lyra-mutedDim">{t('missionSecurityScore.eyebrow')}</p>
          <h2 id="mission-security-score-title" className="mt-1 font-display text-xl font-medium text-lyra-text">{t('missionSecurityScore.title')}</h2>
          <p className="mt-1 max-w-3xl text-xs text-lyra-muted">{t('missionSecurityScore.subtitle')}</p>
        </div>
        <div className={`rounded-lg border px-4 py-3 text-right ${levelClass(result.level)}`}>
          <p className="text-[10px] tracking-[0.12em]">{t('missionSecurityScore.riskScore')}</p>
          <p className="mt-1 font-mono text-3xl">{result.score}/100</p>
          <p className="mt-1 text-[10px]">{t(`missionSecurityScore.level.${result.level}`)}</p>
        </div>
      </div>

      <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-6">
        {result.domains.map(({ domain, score }) => (
          <article key={domain} className="rounded border border-lyra-borderSoft bg-lyra-bg/20 p-3">
            <div className="flex items-center justify-between gap-2">
              <p className="text-[10px] text-lyra-mutedDim">{t(`missionSecurityScore.domain.${domain}`)}</p>
              <span className="font-mono text-xs text-lyra-text">{score}</span>
            </div>
            <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-lyra-borderSoft" aria-label={t('missionSecurityScore.domainAria', { domain: t(`missionSecurityScore.domain.${domain}`), score })}>
              <div className="h-full rounded-full bg-current text-lyra-gold" style={{ width: `${Math.min(100, score)}%` }} />
            </div>
          </article>
        ))}
      </div>

      <div className="mt-5 grid gap-4 lg:grid-cols-[1fr_0.42fr]">
        <div className="rounded border border-lyra-borderSoft bg-lyra-bg/20 p-4">
          <h3 className="text-xs font-medium text-lyra-text">{t('missionSecurityScore.drivers')}</h3>
          {result.factors.length ? (
            <ul className="mt-3 grid gap-2 sm:grid-cols-2">
              {result.factors.map((factor, index) => (
                <FactorRow key={`${factor.key}-${index}`} factor={factor} label={t(`missionSecurityScore.factor.${factor.key}`)} />
              ))}
            </ul>
          ) : (
            <p className="mt-3 text-xs text-lyra-muted">{t('missionSecurityScore.noDrivers')}</p>
          )}
        </div>
        <div className="rounded border border-lyra-borderSoft bg-lyra-bg/20 p-4">
          <h3 className="text-xs font-medium text-lyra-text">{t('missionSecurityScore.coverage')}</h3>
          <p className="mt-3 font-mono text-2xl text-lyra-text">{result.availableSources}/{result.totalSources}</p>
          <p className="mt-1 text-[11px] text-lyra-muted">{t('missionSecurityScore.coverageHint')}</p>
        </div>
      </div>

      <p className="mt-4 text-[10px] text-lyra-mutedDim">{t('missionSecurityScore.disclaimer')}</p>
    </section>
  )
}
