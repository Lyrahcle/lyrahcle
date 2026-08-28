import SeverityBadge from './SeverityBadge.jsx'
import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'

export default function ThreatIntelPanel({ page, onPrev, onNext }) {
  const cves = page.items

  if (!cves.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">No space-related CVEs synced yet.</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">
          Synced automatically from the NVD every 24h across several space-related keywords.
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
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                CVE
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Severity
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Score
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Published
              </th>
              <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                Description
              </th>
            </tr>
          </thead>
          <tbody>
            {cves.map((c) => (
              <tr key={c.cve_id} className="border-b border-lyra-borderSoft/60 last:border-0">
                <td className="py-3 pr-4 font-mono text-xs text-lyra-goldBright">
                  <a
                    href={`https://nvd.nist.gov/vuln/detail/${c.cve_id}`}
                    target="_blank"
                    rel="noreferrer"
                    className="hover:underline"
                  >
                    {c.cve_id}
                  </a>
                </td>
                <td className="py-3 pr-4"><SeverityBadge severity={c.severity} /></td>
                <td className="py-3 pr-4 font-mono text-lyra-text">
                  {c.score ? c.score.toFixed(1) : '—'}
                </td>
                <td className="py-3 pr-4 font-mono text-xs text-lyra-mutedDim">
                  {c.published ? new Date(c.published).toLocaleDateString() : '—'}
                </td>
                <td className="max-w-md py-3 text-xs text-lyra-muted">{c.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
      <Pagination total={page.total} limit={page.limit} offset={page.offset} onPrev={onPrev} onNext={onNext} />
    </div>
  )
}
