import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'
import StatusPill from './StatusPill.jsx'
import OrbitRings from './OrbitRings.jsx'

export default function Hero({ backendStatus, dbStatus, sdrStatus }) {
  return (
    <div className="relative overflow-hidden border-b border-lyra-borderSoft">
      <OrbitRings />

      <div className="relative mx-auto max-w-6xl px-8 py-16 text-center">
        <div className="mx-auto mb-6 h-16 w-16">
          <LogoMark className="h-16 w-16" />
        </div>

        <h1 className="font-display text-3xl font-semibold text-lyra-goldBright">Lyrahcle</h1>
        <p className="mt-2 font-mono text-[11px] uppercase tracking-[0.3em] text-lyra-muted">
          Constellation Analytics
        </p>
        <p className="mx-auto mt-4 max-w-md text-sm italic text-lyra-mutedDim">
          Never Lies. Mission telemetry for space infrastructure security.
        </p>

        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <StatusPill label={`Backend · ${backendStatus}`} tone={toneFor(backendStatus)} />
          <StatusPill label={`Database · ${dbStatus}`} tone={toneFor(dbStatus)} />
          <StatusPill label={`SDR agent · ${sdrStatus}`} tone={toneFor(sdrStatus)} />
          <Clock />
        </div>
      </div>
    </div>
  )
}

function toneFor(status) {
  if (status === 'online' || status === 'live') return 'good'
  if (status === 'idle' || status === 'checking') return 'warn'
  if (status === 'offline') return 'bad'
  return 'idle'
}
