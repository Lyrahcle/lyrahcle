import TableScroll from './TableScroll.jsx'

export default function SatellitesPanel({ satellites }) {
  if (!satellites.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">No satellites tracked yet.</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">
          Register a TLE via POST /api/v1/satellites to begin orbital tracking.
        </p>
      </div>
    )
  }

  return (
    <TableScroll>
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-lyra-borderSoft">
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Satellite
            </th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              NORAD ID
            </th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Latitude
            </th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Longitude
            </th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              Altitude
            </th>
          </tr>
        </thead>
        <tbody>
          {satellites.map((s) => (
            <tr key={s.id} className="border-b border-lyra-borderSoft/60 last:border-0">
              <td className="py-3 text-lyra-goldBright">{s.name}</td>
              <td className="py-3 font-mono text-xs text-lyra-mutedDim">{s.norad_id}</td>
              <td className="py-3 font-mono text-lyra-text">
                {s.position ? `${s.position.latitude_deg.toFixed(2)}°` : '—'}
              </td>
              <td className="py-3 font-mono text-lyra-text">
                {s.position ? `${s.position.longitude_deg.toFixed(2)}°` : '—'}
              </td>
              <td className="py-3 font-mono text-lyra-text">
                {s.position ? `${s.position.altitude_km.toFixed(0)} km` : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </TableScroll>
  )
}
