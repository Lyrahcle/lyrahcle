import SignalBar from './SignalBar.jsx'

export default function RfPanel({ readings }) {
  if (!readings.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">No RF readings yet.</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">
          Start an sdr-agent to begin streaming spectrum data.
        </p>
      </div>
    )
  }

  return (
    <table className="w-full text-left text-sm">
      <thead>
        <tr className="border-b border-lyra-borderSoft">
          <th className="pb-2 text-[10px] font-normal uppercase tracking-wider text-lyra-mutedDim">
            Agent
          </th>
          <th className="pb-2 text-[10px] font-normal uppercase tracking-wider text-lyra-mutedDim">
            Frequency
          </th>
          <th className="pb-2 text-[10px] font-normal uppercase tracking-wider text-lyra-mutedDim">
            Signal
          </th>
          <th className="pb-2 text-[10px] font-normal uppercase tracking-wider text-lyra-mutedDim">
            Captured
          </th>
        </tr>
      </thead>
      <tbody>
        {readings.map((r, i) => (
          <tr key={i} className="border-b border-lyra-borderSoft/60 last:border-0">
            <td className="py-2.5 font-mono text-xs text-lyra-goldBright">{r.agent_id}</td>
            <td className="py-2.5 font-mono text-lyra-text">
              {(r.frequency_hz / 1e6).toFixed(3)} MHz
            </td>
            <td className="py-2.5"><SignalBar dbm={r.power_dbm} /></td>
            <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">
              {new Date(r.captured_at).toLocaleTimeString()}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
