import type { ReactNode } from 'react'

export function HudRail({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={`flex items-center gap-0.5 rounded-full border border-lyra-borderSoft/80 bg-[#080a0d]/78 p-1 shadow-lg shadow-black/40 backdrop-blur-md ${className}`}
    >
      {children}
    </div>
  )
}

export function HudButton({
  label,
  onClick,
  active = false,
  children,
}: {
  label: string
  onClick: () => void
  active?: boolean
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={active}
      title={label}
      onClick={onClick}
      className={`flex h-7 w-7 items-center justify-center rounded-full transition ${
        active
          ? 'bg-lyra-gold/18 text-lyra-goldBright'
          : 'text-lyra-mutedDim hover:bg-lyra-gold/10 hover:text-lyra-goldBright'
      }`}
    >
      {children}
    </button>
  )
}

export function HudChip({
  label,
  active,
  onClick,
  tone,
}: {
  label: string
  active: boolean
  onClick: () => void
  tone?: string
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[10px] transition ${
        active
          ? 'border-lyra-gold/40 bg-lyra-gold/10 text-lyra-goldBright'
          : 'border-lyra-borderSoft/70 text-lyra-mutedDim hover:border-lyra-border hover:text-lyra-muted'
      }`}
    >
      {tone && <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: tone }} />}
      {label}
    </button>
  )
}

export function HudDivider() {
  return <span className="mx-0.5 h-4 w-px bg-lyra-borderSoft/80" aria-hidden="true" />
}
