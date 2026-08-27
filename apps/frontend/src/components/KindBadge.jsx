const colors = {
  domain: 'text-lyra-good border-lyra-good/30 bg-lyra-good/10',
  ip: 'text-lyra-goldBright border-lyra-gold/30 bg-lyra-gold/10',
  satellite: 'text-lyra-warn border-lyra-warn/30 bg-lyra-warn/10',
  ground_station: 'text-lyra-bad border-lyra-bad/30 bg-lyra-bad/10',
  firmware: 'text-lyra-muted border-lyra-border bg-lyra-border/30',
}

export default function KindBadge({ kind }) {
  const cls = colors[kind] ?? colors.firmware
  return (
    <span className={`rounded border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide ${cls}`}>
      {kind}
    </span>
  )
}
