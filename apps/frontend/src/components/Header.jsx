import LogoMark from './LogoMark.jsx'

export default function Header() {
  return (
    <header className="border-b border-lyra-borderSoft bg-lyra-bgSoft/60 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-6 py-4 sm:px-8">
        <LogoMark className="h-6 w-6" />
        <span className="font-display text-sm font-medium text-lyra-goldBright">Lyrahcle</span>
      </div>
    </header>
  )
}
