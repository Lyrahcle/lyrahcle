import { useEffect, useState } from 'react'

const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone
const zoneLabelFormatter = new Intl.DateTimeFormat([], {
  timeZoneName: 'shortOffset',
})

function currentZoneLabel(date) {
  const part = zoneLabelFormatter.formatToParts(date).find((p) => p.type === 'timeZoneName')
  return part ? part.value.replace('GMT', 'UTC') : timeZone
}

export default function Clock() {
  const [now, setNow] = useState(new Date())

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(id)
  }, [])

  const time = now.toLocaleTimeString([], { hour12: false })

  return (
    <span className="font-mono text-xs text-lyra-muted" title={timeZone}>
      {time} <span className="text-lyra-mutedDim">{currentZoneLabel(now)}</span>
    </span>
  )
}
