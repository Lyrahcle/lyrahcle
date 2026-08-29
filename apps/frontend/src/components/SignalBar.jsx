function strengthRatio(dbm) {
  const min = -100
  const max = -30
  const clamped = Math.min(Math.max(dbm, min), max)
  return (clamped - min) / (max - min)
}

function toneFor(ratio) {
  if (ratio > 0.66) return 'bg-lyra-good'
  if (ratio > 0.33) return 'bg-lyra-warn'
  return 'bg-lyra-bad'
}

export default function SignalBar({ dbm }) {
  const ratio = strengthRatio(dbm)
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-20 overflow-hidden rounded-full bg-lyra-border">
        <div
          className={`h-full rounded-full ${toneFor(ratio)}`}
          style={{ width: `${ratio * 100}%` }}
        />
      </div>
      <span className="font-mono text-xs text-lyra-text">{dbm.toFixed(1)} dBm</span>
    </div>
  )
}
