const colors = {
  CRITICAL: 'text-lyra-bad border-lyra-bad/30 bg-lyra-bad/10',
  HIGH: 'text-lyra-bad border-lyra-bad/30 bg-lyra-bad/10',
  MEDIUM: 'text-lyra-warn border-lyra-warn/30 bg-lyra-warn/10',
  LOW: 'text-lyra-good border-lyra-good/30 bg-lyra-good/10',
  UNKNOWN: 'text-lyra-muted border-lyra-border bg-lyra-border/30',
}

export default function SeverityBadge({ severity }) {
  const key = (severity || 'unknown').toUpperCase()
  const cls = colors[key] ?? colors.UNKNOWN
  return (
    <span className={`rounded border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide ${cls}`}>
      {key}
    </span>
  )
}
