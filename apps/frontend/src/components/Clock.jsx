import { useEffect, useState } from 'react'

export default function Clock() {
  const [now, setNow] = useState(new Date())

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(id)
  }, [])

  const utc = now.toUTCString().split(' ')[4]

  return (
    <span className="font-mono text-xs text-lyra-muted">
      {utc} <span className="text-lyra-mutedDim">UTC</span>
    </span>
  )
}
