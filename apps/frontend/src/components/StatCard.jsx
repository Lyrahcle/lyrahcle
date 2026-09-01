const accents = {
  gold: 'from-lyra-gold to-transparent',
  good: 'from-lyra-good to-transparent',
  warn: 'from-lyra-warn to-transparent',
  bad: 'from-lyra-bad to-transparent',
  neutral: 'from-lyra-muted to-transparent',
}

export default function StatCard({ icon: Icon, label, value, accent = 'gold' }) {
  return (
    <div className="relative overflow-hidden rounded-lg border border-lyra-border bg-lyra-panel/50 px-5 py-4">
      <div className={`absolute inset-x-0 top-0 h-px bg-gradient-to-r ${accents[accent]}`} />
      <div className="flex items-center justify-between">
        <p className="text-[10px] uppercase tracking-[0.15em] text-lyra-muted">{label}</p>
        {Icon && <Icon className="h-3.5 w-3.5 text-lyra-mutedDim" strokeWidth={1.5} />}
      </div>
      <p className="mt-2 font-mono text-2xl font-medium text-lyra-text">{value}</p>
    </div>
  )
}
