import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'

export default function Header({ onLogout, onHome }) {
  return (
    <header className="sticky top-0 z-40 border-b border-lyra-borderSoft bg-lyra-bg/85 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-3 px-6 py-3.5 sm:px-8">
        <button
          onClick={onHome}
          className="flex items-center gap-2.5 rounded outline-none focus-visible:ring-1 focus-visible:ring-lyra-gold"
        >
          <LogoMark className="h-6 w-6" />
          <span className="font-display text-[15px] font-medium tracking-tight text-lyra-goldBright">
            Lyrahcle
          </span>
        </button>
        <div className="flex items-center gap-4">
          <Clock />
          {onLogout && (
            <button
              onClick={onLogout}
              className="rounded border border-lyra-border px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
            >
              Sign out
            </button>
          )}
        </div>
      </div>
    </header>
  )
}
