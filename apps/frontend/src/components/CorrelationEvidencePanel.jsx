import SeverityBadge from './SeverityBadge.jsx'
import { useI18n } from '../lib/I18nContext'

function percent(value) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric)) return '—'
  return `${Math.round(Math.max(0, Math.min(1, numeric)) * 100)}%`
}

function asArray(value) {
  return Array.isArray(value) ? value : []
}

function domainLabel(value) {
  return String(value || '').replaceAll('_', ' ')
}

function contextEvidence(evidence, reference) {
  return evidence.find((item) => item?.evidence_ref === reference)
}

function ContextItems({ item, title, emptyLabel }) {
  const { t } = useI18n()
  const rows = asArray(item?.metadata?.items)
  if (!item) return null

  return (
    <div className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/20 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h5 className="font-mono text-[11px] tracking-wider text-lyra-goldBright">{title}</h5>
        {(item.metadata?.candidate_only || item.metadata?.context_only) && (
          <span className="rounded border border-lyra-border px-2 py-1 font-mono text-[10px] text-lyra-muted">
            {item.metadata?.context_only ? t('correlation.contextOnly') : t('correlation.candidateOnly')}
          </span>
        )}
      </div>
      {!rows.length ? (
        <p className="mt-2 text-xs text-lyra-muted">{emptyLabel}</p>
      ) : (
        <div className="mt-2 grid gap-2">
          {rows.slice(0, 6).map((row, index) => (
            <div key={row.id || row.alert_id || `${item.id}-${index}`} className="rounded border border-lyra-borderSoft/70 px-3 py-2 text-xs">
              {row.event_type && <span className="font-mono text-lyra-goldBright">{domainLabel(row.event_type)}</span>}
              {row.action && <span className="font-mono text-lyra-goldBright">{row.action}</span>}
              {row.cve_id && <span className="font-mono text-lyra-goldBright">{row.cve_id}</span>}
              {row.asset_value && <p className="mt-1 break-all text-lyra-text">{row.asset_value}</p>}
              {row.source && <p className="mt-1 text-lyra-muted">{row.source}</p>}
              {row.event_time && <p className="mt-1 font-mono text-[10px] text-lyra-mutedDim">{new Date(row.event_time).toLocaleString()}</p>}
              {row.occurred_at && <p className="mt-1 font-mono text-[10px] text-lyra-mutedDim">{new Date(row.occurred_at).toLocaleString()}</p>}
              <div className="mt-1 flex flex-wrap gap-2">
                {row.severity && <SeverityBadge severity={row.severity} />}
                {row.confidence !== undefined && <span className="font-mono text-[10px] text-lyra-muted">{t('correlation.confidence')}: {percent(row.confidence)}</span>}
              </div>
            </div>
          ))}
          {rows.length > 6 && <p className="text-[11px] text-lyra-mutedDim">{t('correlation.moreItems', { count: rows.length - 6 })}</p>}
        </div>
      )}
    </div>
  )
}

export default function CorrelationEvidencePanel({ evidence = [] }) {
  const { t } = useI18n()
  const summary = evidence.find((item) => item?.evidence_ref === 'cross-domain-v9.4' || item?.metadata?.policy === 'explainable_candidate_only')
  if (!summary) return null

  const metadata = summary.metadata || {}
  const corroborating = asArray(metadata.corroborating_domains)
  const contextOnly = asArray(metadata.context_only_domains)
  const reasons = asArray(metadata.reasons)
  const orbitalOperations = contextEvidence(evidence, 'orbital-operations-v9.5')
  const assetAlerts = contextEvidence(evidence, 'unacknowledged-7d')
  const agentAudit = evidence.find((item) => item?.label === 'Recent agent audit activity')

  return (
    <section aria-label={t('correlation.title')} className="grid gap-3 rounded border border-lyra-gold/30 bg-lyra-gold/5 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="font-mono text-[10px] tracking-[0.18em] text-lyra-gold">{t('correlation.eyebrow')}</p>
          <h4 className="mt-1 text-base text-lyra-text">{t('correlation.title')}</h4>
          <p className="mt-1 max-w-3xl text-xs text-lyra-muted">{t('correlation.explain')}</p>
        </div>
        <div className="text-right">
          <p className="font-mono text-2xl text-lyra-goldBright">{percent(metadata.confidence)}</p>
          <p className="font-mono text-[10px] text-lyra-mutedDim">{t('correlation.confidence')}</p>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <div className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-3">
          <p className="font-mono text-[10px] tracking-wider text-lyra-mutedDim">{t('correlation.candidateSeverity')}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            {metadata.candidate_severity ? <SeverityBadge severity={metadata.candidate_severity} /> : <span className="text-xs text-lyra-muted">{t('correlation.none')}</span>}
            <span className="rounded border border-lyra-border px-2 py-1 font-mono text-[10px] text-lyra-muted">
              {metadata.severity_applied ? t('correlation.applied') : t('correlation.notApplied')}
            </span>
          </div>
        </div>
        <div className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-3">
          <p className="font-mono text-[10px] tracking-wider text-lyra-mutedDim">{t('correlation.policy')}</p>
          <p className="mt-2 font-mono text-xs text-lyra-text">{metadata.policy || '—'}</p>
        </div>
      </div>

      <div className="grid gap-3 lg:grid-cols-2">
        <div>
          <p className="mb-2 font-mono text-[10px] tracking-wider text-lyra-mutedDim">{t('correlation.corroborating')}</p>
          <div className="flex flex-wrap gap-2">
            {corroborating.length ? corroborating.map((domain) => (
              <span key={domain} className="rounded border border-lyra-ok/40 bg-lyra-ok/5 px-2 py-1 font-mono text-[10px] text-lyra-ok">{domainLabel(domain)}</span>
            )) : <span className="text-xs text-lyra-muted">{t('correlation.none')}</span>}
          </div>
        </div>
        <div>
          <p className="mb-2 font-mono text-[10px] tracking-wider text-lyra-mutedDim">{t('correlation.contextDomains')}</p>
          <div className="flex flex-wrap gap-2">
            {contextOnly.length ? contextOnly.map((domain) => (
              <span key={domain} className="rounded border border-lyra-border px-2 py-1 font-mono text-[10px] text-lyra-muted">{domainLabel(domain)}</span>
            )) : <span className="text-xs text-lyra-muted">{t('correlation.none')}</span>}
          </div>
        </div>
      </div>

      {!!reasons.length && (
        <div>
          <p className="mb-2 font-mono text-[10px] tracking-wider text-lyra-mutedDim">{t('correlation.reasons')}</p>
          <ul className="grid gap-1.5 text-xs text-lyra-text">
            {reasons.map((reason) => <li key={reason} className="flex gap-2"><span aria-hidden="true" className="text-lyra-gold">•</span><span>{reason}</span></li>)}
          </ul>
        </div>
      )}

      <div className="grid gap-3 xl:grid-cols-3">
        <ContextItems item={orbitalOperations} title={t('correlation.orbitalOperations')} emptyLabel={t('correlation.noOrbitalOperations')} />
        <ContextItems item={agentAudit} title={t('correlation.agentAudit')} emptyLabel={t('correlation.noAgentAudit')} />
        <ContextItems item={assetAlerts} title={t('correlation.assetAlerts')} emptyLabel={t('correlation.noAssetAlerts')} />
      </div>
    </section>
  )
}
