import KindBadge from './KindBadge.jsx'
import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'

export default function AssetsTable({ page, onPrev, onNext }) {
  const assets = page.items

  if (!assets.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">No assets tracked yet.</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">
          Add one manually above, or wait for an automated recon scan (planned).
        </p>
      </div>
    )
  }

  return (
    <div>
      <TableScroll>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-lyra-borderSoft">
              <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Kind
              </th>
              <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Value
              </th>
              <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Program
              </th>
            </tr>
          </thead>
          <tbody>
            {assets.map((a) => (
              <tr key={a.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                <td className="py-3"><KindBadge kind={a.kind} /></td>
                <td className="py-3 font-mono text-lyra-text">{a.value}</td>
                <td className="py-3 font-mono text-xs text-lyra-mutedDim">{a.program_id}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
      <Pagination total={page.total} limit={page.limit} offset={page.offset} onPrev={onPrev} onNext={onNext} />
    </div>
  )
}
