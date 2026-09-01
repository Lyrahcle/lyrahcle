import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'
import { useI18n } from '../lib/I18nContext'

export default function AuditLogPanel({ page, onPrev, onNext }) {
  const { t } = useI18n()
  const entries = page.items

  if (!entries.length) {
    return <p className="text-sm text-lyra-muted">{t('auditLog.empty')}</p>
  }

  return (
    <div>
      <TableScroll>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-lyra-borderSoft">
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('auditLog.when')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('auditLog.action')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('auditLog.actor')}
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                {t('auditLog.ip')}
              </th>
            </tr>
          </thead>
          <tbody>
            {entries.map((e) => (
              <tr key={e.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">
                  {new Date(e.occurred_at).toLocaleString()}
                </td>
                <td className="py-2.5 font-mono text-xs text-lyra-goldBright">{e.action}</td>
                <td className="py-2.5 font-mono text-xs text-lyra-text">
                  {e.actor_type}:{e.actor_id.slice(0, 12)}
                </td>
                <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">{e.ip || '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
      <Pagination total={page.total} limit={page.limit} offset={page.offset} onPrev={onPrev} onNext={onNext} />
    </div>
  )
}
