import { useState } from 'react'
import SeverityBadge from './SeverityBadge.jsx'
import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'
import { acknowledgeAssetAlert } from '../lib/api'
import { hasPermission } from '../lib/permissions'
import { useI18n } from '../lib/I18nContext'
import StatusState from './StatusState.tsx'
import { useToast } from './ToastProvider.tsx'
import EvidenceBadge from './EvidenceBadge.tsx'

export default function AssetAlertsPanel({ page, onPrev, onNext, onChanged }) {
  const { t } = useI18n()
  const { notify } = useToast()
  const [error, setError] = useState(null)
  const [ackingId, setAckingId] = useState(null)
  const alerts = page.items

  async function handleAcknowledge(id) {
    setError(null)
    setAckingId(id)
    try {
      await acknowledgeAssetAlert(id)
      onChanged()
      notify({ title: t('ux.notifications.alertAcknowledged'), tone: 'success', dedupeKey: `asset-alert-${id}` })
    } catch (err) {
      setError(err.message)
    } finally {
      setAckingId(null)
    }
  }

  if (!alerts.length) {
    return (
      <StatusState kind="empty" title={t('assetAlerts.empty')} message={t('assetAlerts.emptyHint')} />
    )
  }

  return (
    <div>
      <div className="mb-4 rounded border border-lyra-warn/25 bg-lyra-warn/5 px-4 py-3">
        <p className="text-sm text-lyra-warn">{t('assetAlerts.contextWarningTitle')}</p>
        <p className="mt-1 text-xs text-lyra-muted">{t('assetAlerts.contextWarning')}</p>
      </div>
      {error && (
        <p role="alert" className="mb-4 text-sm text-lyra-bad">
          {error}
        </p>
      )}

      <TableScroll>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-lyra-borderSoft">
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.asset')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                CVE
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.severity')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.matchedVia')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.assessment')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.status')}
              </th>
              <th scope="col" className="pb-2" />
            </tr>
          </thead>
          <tbody>
            {alerts.map((a) => (
              <tr key={a.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                <td className="py-3 pr-4">
                  <span className="font-mono text-xs text-lyra-goldBright">{a.asset_value}</span>
                  <span className="ml-2 rounded-full border border-lyra-border px-2 py-0.5 text-[10px] tracking-wider text-lyra-mutedDim">
                    {a.asset_kind}
                  </span>
                </td>
                <td className="py-3 pr-4 font-mono text-xs text-lyra-text">
                  <a
                    href={`https://nvd.nist.gov/vuln/detail/${a.cve_id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:underline"
                  >
                    {a.cve_id}
                  </a>
                </td>
                <td className="py-3 pr-4"><SeverityBadge severity={a.severity} /></td>
                <td className="py-3 pr-4">
                  <p className="font-mono text-xs text-lyra-mutedDim">{a.matched_keyword}</p>
                  <p className="mt-1 text-[10px] text-lyra-mutedDim">{t(`assetAlerts.matchMethod.${a.match_method || 'thematic_asset_kind'}`)}</p>
                </td>
                <td className="py-3 pr-4">
                  <EvidenceBadge kind={a.context_only === false ? 'confirmed' : 'context'} />
                  <p className="mt-1 text-[10px] text-lyra-mutedDim">
                    {t(`assetAlerts.applicability.${a.applicability_status || 'unknown'}`)} · {t('assetAlerts.confidence', { value: a.confidence ?? 20 })}
                  </p>
                </td>
                <td className="py-3 pr-4">
                  {a.acknowledged ? (
                    <span className="font-mono text-[11px] tracking-wider text-lyra-good">
                      {t('assetAlerts.acknowledged')}
                    </span>
                  ) : (
                    <span className="font-mono text-[11px] tracking-wider text-lyra-warn">
                      {t('assetAlerts.new')}
                    </span>
                  )}
                </td>
                <td className="py-3 text-right">
                  {!a.acknowledged && hasPermission('alerts.manage') && (
                    <button
                      onClick={() => handleAcknowledge(a.id)}
                      disabled={ackingId === a.id}
                      className="font-mono text-[11px] tracking-wide text-lyra-gold outline-none hover:underline disabled:opacity-50"
                    >
                      {ackingId === a.id ? t('common.saving') : t('assetAlerts.acknowledge')}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
      <Pagination total={page.total} limit={page.limit} offset={page.offset} onPrev={onPrev} onNext={onNext} />
    </div>
  )
}
