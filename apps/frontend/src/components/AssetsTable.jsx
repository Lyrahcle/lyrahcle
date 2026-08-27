export default function AssetsTable({ assets }) {
  if (!assets.length) {
    return (
      <p className="rounded-lg border border-dashed border-lyra-panel px-5 py-8 text-center text-sm text-lyra-muted">
        No assets discovered yet.
      </p>
    )
  }

  return (
    <table className="w-full text-left text-sm">
      <thead>
        <tr className="border-b border-lyra-panel text-lyra-muted">
          <th className="py-2 font-normal">Kind</th>
          <th className="py-2 font-normal">Value</th>
          <th className="py-2 font-normal">Program</th>
        </tr>
      </thead>
      <tbody>
        {assets.map((a) => (
          <tr key={a.id} className="border-b border-lyra-panel/60">
            <td className="py-2 text-lyra-goldLight">{a.kind}</td>
            <td className="py-2">{a.value}</td>
            <td className="py-2 text-lyra-muted">{a.program_id}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}
