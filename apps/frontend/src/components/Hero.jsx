import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'
import StatusPill from './StatusPill.jsx'
import OrbitRings from './OrbitRings.jsx'

export default function Hero({ backendStatus, dbStatus, sdrStatus }) {
  return (
    <div className="relative overflow-hidden border-b border-lyra-borderSoft">
      <OrbitRings />

      <div className="relative mx-auto max-w-7xl px-6 py-14 text-center sm:px-8 sm:py-20">
        <div className="mx-auto mb-7 h-16 w-16 sm:h-20 sm:w-20">
          <LogoMark className="h-16 w-16 sm:h-20 sm:w-20" />
        </div>

        <h1 className="font-display text-3xl font-semibold text-lyra-goldBright sm:text-4xl">
          Lyrahcle
        </h1>
        <p className="mt-2 font-mono text-xs uppercase tracking-[0.3em] text-lyra-muted">
          Constellation Analytics
        </p>
        <p className="mx-auto mt-4 max-w-lg text-base italic text-lyra-mutedDim">
          Never Lies. Mission telemetry for space infrastructure security.
        </p>

        <div className="mt-9 flex flex-wrap items-center justify-center gap-3">
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
