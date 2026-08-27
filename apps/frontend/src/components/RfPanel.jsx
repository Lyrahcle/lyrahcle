export default function RfPanel({ readings }) {
  if (!readings.length) {
    return (
      <p className="rounded-lg border border-dashed border-lyra-panel px-5 py-8 text-center text-sm text-lyra-muted">
        No RF readings yet. Start the sdr-agent to see live spectrum data.
      </p>
    )
  }

  return (
    <table className="w-full text-left text-sm">
      <thead>
        <tr className="border-b border-lyra-panel text-lyra-muted">
          <th className="py-2 font-normal">Agent</th>
          <th className="py-2 font-normal">Frequency</th>
          <th className="py-2 font-normal">Power</th>
          <th className="py-2 font-normal">Captured</th>
        </tr>
      </thead>
      <tbody>
        {readings.map((r, i) => (
          <tr key={i} className="border-b border-lyra-panel/60">
            <td className="py-2 text-lyra-goldLight">{r.agent_id}</td>
            <td className="py-2">{(r.frequency_hz / 1e6).toFixed(3)} MHz</td>
            <td className="py-2">{r.power_dbm.toFixed(1)} dBm</td>
            <td className="py-2 text-lyra-muted">{new Date(r.captured_at).toLocaleTimeString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
