export default function StatCard({ label, value }) {
  return (
    <div className="rounded-lg border border-lyra-panel bg-lyra-panel/40 px-5 py-4">
      <p className="text-xs uppercase tracking-wide text-lyra-muted">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-lyra-goldLight">{value}</p>
    </div>
  )
}
