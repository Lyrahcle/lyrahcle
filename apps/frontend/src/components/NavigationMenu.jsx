import { useEffect, useRef, useState } from 'react'
import {
  Menu,
  Orbit,
  Boxes,
  RadioTower,
  ShieldAlert,
  Radar,
  Binary,
  Cable,
  UsersRound,
  Settings,
  Building2,
  ChevronDown,
} from 'lucide-react'

const ICONS = {
  orbital: Orbit,
  assets: Boxes,
  signal: RadioTower,
  incidents: ShieldAlert,
  intel: Radar,
  decoder: Binary,
  connectivity: Cable,
  admin: UsersRound,
  platform: Building2,
  settings: Settings,
}

export default function NavigationMenu({ items, active, onChange }) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef(null)
  const activeItem = items.find((item) => item.id === active) ?? items[0]

  useEffect(() => {
    function handlePointer(event) {
      if (!rootRef.current?.contains(event.target)) setOpen(false)
    }
    function handleKey(event) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', handlePointer)
    document.addEventListener('keydown', handleKey)
    return () => {
      document.removeEventListener('pointerdown', handlePointer)
      document.removeEventListener('keydown', handleKey)
    }
  }, [])

  function select(id) {
    onChange(id)
    setOpen(false)
  }

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Open navigation"
        onClick={() => setOpen((value) => !value)}
        className="flex h-9 items-center gap-2 rounded border border-lyra-border bg-lyra-panel/70 px-2.5 text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
      >
        <Menu className="h-4 w-4" strokeWidth={1.8} />
        <span className="hidden max-w-28 truncate font-mono text-[10px] uppercase tracking-[0.16em] sm:inline">
          {activeItem?.label}
        </span>
        <ChevronDown className={`hidden h-3.5 w-3.5 transition-transform sm:block ${open ? 'rotate-180' : ''}`} strokeWidth={1.8} />
      </button>

      {open && (
        <div
          role="menu"
          aria-label="Primary navigation"
          className="absolute left-0 top-full z-[70] mt-2 w-[min(19rem,calc(100vw-2rem))] overflow-hidden rounded-xl border border-lyra-border bg-lyra-panel p-2 shadow-2xl backdrop-blur-xl"
        >
          <div className="px-2 pb-2 pt-1 font-mono text-[9px] uppercase tracking-[0.24em] text-lyra-mutedDim">
            Lyrahcle navigation
          </div>
          <div className="grid gap-1">
            {items.map((item) => {
              const Icon = ICONS[item.id] ?? Orbit
              const selected = item.id === active
              return (
                <button
                  key={item.id}
                  type="button"
                  role="menuitem"
                  aria-current={selected ? 'page' : undefined}
                  onClick={() => select(item.id)}
                  className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left outline-none transition-colors focus-visible:ring-1 focus-visible:ring-lyra-gold ${
                    selected
                      ? 'bg-lyra-gold/10 text-lyra-goldBright'
                      : 'text-lyra-muted hover:bg-white/[0.035] hover:text-lyra-text'
                  }`}
                >
                  <span className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-md border ${selected ? 'border-lyra-gold/30 bg-lyra-gold/10' : 'border-lyra-borderSoft bg-black/10'}`}>
                    <Icon className="h-4 w-4" strokeWidth={1.7} />
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium">{item.label}</span>
                    {item.id === 'orbital' && (
                      <span className="mt-0.5 block font-mono text-[9px] uppercase tracking-[0.16em] text-lyra-mutedDim">Default dashboard</span>
                    )}
                  </span>
                </button>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
