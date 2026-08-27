export default function Header() {
  return (
    <header className="flex items-center justify-between border-b border-lyra-panel px-8 py-5">
      <div className="flex items-center gap-3">
        <img src="/logo.jpeg" alt="Lyrahcle" className="h-9 w-auto rounded" />
        <div>
          <h1 className="font-display text-xl text-lyra-goldLight leading-none">Lyrahcle</h1>
          <p className="text-xs uppercase tracking-widest text-lyra-muted">Constellation Analytics</p>
        </div>
      </div>
      <span className="text-xs text-lyra-muted italic">Never Lies.</span>
    </header>
  )
}
