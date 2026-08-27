const tones = {
  good: 'bg-lyra-good',
  warn: 'bg-lyra-warn',
  bad: 'bg-lyra-bad',
  idle: 'bg-lyra-mutedDim',
}

export default function StatusPill({ label, tone = 'idle' }) {
  return (
    <div className="flex items-center gap-2 rounded-full border border-lyra-border bg-lyra-panel/60 px-3.5 py-1.5">
      <span className={`h-1.5 w-1.5 rounded-full ${tones[tone]}`} />
      <span className="font-mono text-xs uppercase tracking-wider text-lyra-muted">{label}</span>
    </div>
  )
}
