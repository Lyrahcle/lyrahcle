import Clock from './Clock.jsx'
import StatusPill from './StatusPill.jsx'

export default function Header({ backendStatus }) {
  const tone = backendStatus === 'online' ? 'good' : backendStatus === 'offline' ? 'bad' : 'idle'

  return (
    <header className="border-b border-lyra-borderSoft bg-lyra-bgSoft/60 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-8 py-5">
        <div className="flex items-center gap-3">
          <div className="relative">
            <div className="absolute inset-0 rounded-full bg-lyra-gold/20 blur-md" />
            <img src="/logo.jpeg" alt="Lyrahcle" className="relative h-9 w-9 rounded-full object-cover" />
          </div>
          <div>
            <h1 className="font-display text-lg font-semibold leading-none text-lyra-goldBright">
              Lyrahcle
            </h1>
            <p className="mt-0.5 text-[10px] uppercase tracking-[0.2em] text-lyra-muted">
              Constellation Analytics
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <Clock />
          <StatusPill label={`Backend · ${backendStatus}`} tone={tone} />
        </div>
      </div>
    </header>
  )
}
