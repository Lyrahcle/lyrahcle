import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'

export default function AuditLogPanel({ page, onPrev, onNext }) {
  const entries = page.items

  if (!entries.length) {
    return <p className="text-sm text-lyra-muted">No audit events recorded yet.</p>
  }

  return (
    <div>
      <TableScroll>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-lyra-borderSoft">
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                When
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Action
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Actor
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                IP
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
