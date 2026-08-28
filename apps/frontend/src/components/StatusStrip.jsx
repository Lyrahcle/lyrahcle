import StatusPill from './StatusPill.jsx'

export default function StatusStrip({ backendStatus, dbStatus, sdrStatus }) {
  return (
    <div className="border-b border-lyra-borderSoft bg-lyra-bgSoft/40">
      <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-2 px-6 py-2.5 sm:px-8">
        <StatusPill label={`Backend · ${backendStatus}`} tone={toneFor(backendStatus)} />
        <StatusPill label={`Database · ${dbStatus}`} tone={toneFor(dbStatus)} />
        <StatusPill label={`SDR agent · ${sdrStatus}`} tone={toneFor(sdrStatus)} />
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
