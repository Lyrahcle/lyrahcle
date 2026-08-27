import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'

export default function Header() {
  return (
    <header className="sticky top-0 z-40 border-b border-lyra-borderSoft bg-lyra-bg/80 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-3 px-6 py-4 sm:px-8">
        <div className="flex items-center gap-3">
          <LogoMark className="h-7 w-7" />
          <span className="font-display text-base font-medium text-lyra-goldBright">
            Lyrahcle
          </span>
        </div>
        <Clock />
      </div>
    </header>
  )
}
