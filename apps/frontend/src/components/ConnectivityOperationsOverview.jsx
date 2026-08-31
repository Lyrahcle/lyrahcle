import { useMemo, useState } from 'react'
import { Activity, Cpu, RadioTower, Search, ShieldAlert, ShieldCheck, WifiOff } from 'lucide-react'
import { useI18n } from '../lib/I18nContext.jsx'

const ONLINE_WINDOW_MS = 5 * 60 * 1000

function mergeAgents(identities = [], legacyKeys = []) {
  const merged = new Map()
  for (const agent of identities) merged.set(agent.agent_id, { ...agent, source: 'mtls' })
  for (const key of legacyKeys) {
    const current = merged.get(key.agent_id)
    if (!current) {
      merged.set(key.agent_id, {
        agent_id: key.agent_id,
        status: key.revoked ? 'revoked' : 'active',
        last_seen_at: key.last_used_at,
        device: key.device,
        source: 'legacy',
      })
    } else if (!current.device && key.device) {
      merged.set(key.agent_id, { ...current, device: key.device })
    }
  }
  return [...merged.values()].sort((a, b) => a.agent_id.localeCompare(b.agent_id))
}

function lastSeenMs(agent) {
  const value = agent.last_seen_at || agent.device?.reported_at
  if (!value) return null
  const parsed = Date.parse(value)
  return Number.isFinite(parsed) ? parsed : null
}

function posture(agent, now) {
  if (agent.status === 'quarantined' || agent.status === 'revoked') return 'attention'
  const integrity = agent.device?.capabilities?.integrity_status
  const compatibility = agent.device?.compatibility
  if (integrity === 'changed' || compatibility === 'unsupported' || compatibility === 'degraded') return 'attention'
  const seen = lastSeenMs(agent)
  if (!seen || now - seen > ONLINE_WINDOW_MS) return 'offline'
  return 'healthy'
}

function badgeClass(state) {
  if (state === 'healthy') return 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
  if (state === 'attention') return 'border-lyra-warn/30 bg-lyra-warn/10 text-lyra-warn'
  return 'border-lyra-border bg-lyra-bgSoft text-lyra-muted'
}

export default function ConnectivityOperationsOverview({ agentIdentities = [], agentKeys = [] }) {
  const { t } = useI18n()
  const [filter, setFilter] = useState('all')
  const [query, setQuery] = useState('')
  const now = Date.now()
  const agents = useMemo(() => mergeAgents(agentIdentities, agentKeys), [agentIdentities, agentKeys])
  const enriched = agents.map((agent) => ({ ...agent, posture: posture(agent, now) }))
  const counts = enriched.reduce((acc, agent) => {
    acc[agent.posture] += 1
    if (agent.device) acc.reported += 1
    return acc
  }, { healthy: 0, attention: 0, offline: 0, reported: 0 })
  const visible = enriched.filter((agent) => {
    if (filter !== 'all' && agent.posture !== filter) return false
    return agent.agent_id.toLowerCase().includes(query.trim().toLowerCase())
  })

  return (
    <section className="grid gap-4" aria-labelledby="connectivity-operations-heading">
      <div>
        <p className="text-[10px] uppercase tracking-[0.18em] text-lyra-mutedDim">{t('connectivity.opsEyebrow')}</p>
        <h1 id="connectivity-operations-heading" className="mt-1 font-display text-2xl font-medium text-lyra-text">{t('connectivity.opsTitle')}</h1>
        <p className="mt-1 max-w-3xl text-sm text-lyra-muted">{t('connectivity.opsHint')}</p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {[
          ['healthy', counts.healthy, Activity],
          ['attention', counts.attention, ShieldAlert],
          ['offline', counts.offline, WifiOff],
          ['reported', counts.reported, Cpu],
        ].map(([key, value, Icon]) => (
          <div key={key} className="rounded-lg border border-lyra-border bg-lyra-panel/50 px-4 py-3">
            <div className="flex items-center justify-between gap-2">
              <p className="text-[10px] uppercase tracking-[0.15em] text-lyra-muted">{t(`connectivity.opsMetric.${key}`)}</p>
              <Icon className="h-3.5 w-3.5 text-lyra-mutedDim" aria-hidden="true" />
            </div>
            <p className="mt-2 font-mono text-2xl text-lyra-text">{value}</p>
          </div>
        ))}
      </div>

      <div className="rounded-lg border border-lyra-border bg-lyra-panel/50 p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-wrap gap-2" role="group" aria-label={t('connectivity.opsFilterLabel')}>
            {['all', 'healthy', 'attention', 'offline'].map((item) => (
              <button
                key={item}
                type="button"
                onClick={() => setFilter(item)}
                aria-pressed={filter === item}
                className={`rounded border px-3 py-1.5 text-xs ${filter === item ? 'border-lyra-gold/40 bg-lyra-gold/10 text-lyra-goldBright' : 'border-lyra-border text-lyra-muted hover:text-lyra-text'}`}
              >
                {t(`connectivity.opsFilter.${item}`)}
              </button>
            ))}
          </div>
          <label className="relative block min-w-0 sm:min-w-64">
            <span className="sr-only">{t('connectivity.opsSearch')}</span>
            <Search className="pointer-events-none absolute left-2.5 top-2.5 h-3.5 w-3.5 text-lyra-mutedDim" aria-hidden="true" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t('connectivity.opsSearch')}
              className="w-full rounded border border-lyra-border bg-lyra-bgSoft py-2 pl-8 pr-3 text-sm text-lyra-text outline-none focus:border-lyra-gold/50"
            />
          </label>
        </div>

        {!visible.length ? (
          <p className="mt-4 rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-4 text-sm text-lyra-muted">{t('connectivity.opsEmpty')}</p>
        ) : (
          <div className="mt-4 grid gap-2">
            {visible.map((agent) => {
              const integrity = agent.device?.capabilities?.integrity_status
              const lastSeen = agent.last_seen_at || agent.device?.reported_at
              return (
                <div key={agent.agent_id} className="grid gap-3 rounded border border-lyra-borderSoft bg-lyra-bg/25 px-3 py-3 md:grid-cols-[1.2fr_.8fr_1fr_1fr] md:items-center">
                  <div className="min-w-0">
                    <p className="truncate font-mono text-xs text-lyra-goldBright">{agent.agent_id}</p>
                    <p className="mt-1 text-[11px] text-lyra-mutedDim">{agent.source === 'legacy' ? t('connectivity.legacyAgentKey') : 'mTLS'}</p>
                  </div>
                  <div>
                    <span className={`inline-flex rounded-full border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider ${badgeClass(agent.posture)}`}>{t(`connectivity.opsState.${agent.posture}`)}</span>
                    <p className="mt-1 text-[11px] text-lyra-mutedDim">{lastSeen ? new Date(lastSeen).toLocaleString() : t('connectivity.opsNeverSeen')}</p>
                  </div>
                  <div className="flex items-start gap-2">
                    <RadioTower className="mt-0.5 h-3.5 w-3.5 shrink-0 text-lyra-mutedDim" aria-hidden="true" />
                    <div className="min-w-0 text-xs">
                      <p className="truncate text-lyra-text">{agent.device?.hardware || agent.device?.product || agent.device?.driver || t('connectivity.noDeviceReported')}</p>
                      <p className="mt-1 text-[11px] text-lyra-mutedDim">{agent.device?.compatibility ? t(`connectivity.compatibility.${agent.device.compatibility}`) : '—'}</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0 text-lyra-mutedDim" aria-hidden="true" />
                    <div className="text-xs">
                      <p className="text-lyra-text">{integrity ? t(`connectivity.integrity.${integrity}`) : t('connectivity.opsIntegrityUnknown')}</p>
                      <p className="mt-1 text-[11px] text-lyra-mutedDim">{t('connectivity.identityStatus')}: {agent.status || '—'}</p>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </section>
  )
}
