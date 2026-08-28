import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'
import AccountMenu from './AccountMenu.jsx'

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
          <span className="hidden font-mono text-[10px] uppercase tracking-[0.25em] text-lyra-mutedDim sm:inline">
            Constellation Analytics
          </span>
        </button>

        <div className="flex items-center gap-4">
          <Clock />
          {onLogout && <AccountMenu onLogout={onLogout} />}
        </div>
      </div>
    </header>
  )
}
