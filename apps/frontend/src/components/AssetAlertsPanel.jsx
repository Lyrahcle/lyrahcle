import { useState } from 'react'
import SeverityBadge from './SeverityBadge.jsx'
import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'
import { acknowledgeAssetAlert } from '../lib/api.js'
import { canWrite } from '../lib/auth.js'
import { useI18n } from '../lib/I18nContext.jsx'

export default function AssetAlertsPanel({ page, onPrev, onNext, onChanged }) {
  const { t } = useI18n()
  const [error, setError] = useState(null)
  const [ackingId, setAckingId] = useState(null)
  const alerts = page.items

  async function handleAcknowledge(id) {
    setError(null)
    setAckingId(id)
    try {
      await acknowledgeAssetAlert(id)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setAckingId(null)
    }
  }

  if (!alerts.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">{t('assetAlerts.empty')}</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">{t('assetAlerts.emptyHint')}</p>
      </div>
    )
  }

  return (
    <div>
      {error && (
        <p role="alert" className="mb-4 text-sm text-lyra-bad">
          {error}
        </p>
      )}

      <TableScroll>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-lyra-borderSoft">
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.asset')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                CVE
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.severity')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                {t('assetAlerts.matchedVia')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
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
                  <span className="ml-2 rounded-full border border-lyra-border px-2 py-0.5 text-[10px] uppercase tracking-wider text-lyra-mutedDim">
                    {a.asset_kind}
                  </span>
                </td>
                <td className="py-3 pr-4 font-mono text-xs text-lyra-text">
                  <a
                    href={`https://nvd.nist.gov/vuln/detail/${a.cve_id}`}
                    target="_blank"
                    rel="noreferrer"
                    className="hover:underline"
                  >
                    {a.cve_id}
                  </a>
                </td>
                <td className="py-3 pr-4"><SeverityBadge severity={a.severity} /></td>
                <td className="py-3 pr-4 font-mono text-xs text-lyra-mutedDim">{a.matched_keyword}</td>
                <td className="py-3 pr-4">
                  {a.acknowledged ? (
                    <span className="font-mono text-[11px] uppercase tracking-wider text-lyra-good">
                      {t('assetAlerts.acknowledged')}
                    </span>
                  ) : (
                    <span className="font-mono text-[11px] uppercase tracking-wider text-lyra-warn">
                      {t('assetAlerts.new')}
                    </span>
                  )}
                </td>
                <td className="py-3 text-right">
                  {!a.acknowledged && canWrite() && (
                    <button
                      onClick={() => handleAcknowledge(a.id)}
                      disabled={ackingId === a.id}
                      className="font-mono text-[11px] uppercase tracking-wide text-lyra-gold outline-none hover:underline disabled:opacity-50"
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
