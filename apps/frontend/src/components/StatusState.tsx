import { AlertTriangle, Inbox, LoaderCircle, ShieldX, WifiOff } from 'lucide-react'

export type StatusStateKind = 'empty' | 'loading' | 'error' | 'offline' | 'forbidden'

interface StatusStateProps {
  kind: StatusStateKind
  title: string
  message?: string
  actionLabel?: string
  onAction?: () => void
  compact?: boolean
}

function iconFor(kind: StatusStateKind) {
  if (kind === 'loading') return LoaderCircle
  if (kind === 'error') return AlertTriangle
  if (kind === 'offline') return WifiOff
  if (kind === 'forbidden') return ShieldX
  return Inbox
}

function accentFor(kind: StatusStateKind): string {
  if (kind === 'error') return 'text-lyra-bad'
  if (kind === 'offline') return 'text-lyra-warn'
  if (kind === 'forbidden') return 'text-lyra-warn'
  if (kind === 'loading') return 'text-lyra-goldBright'
  return 'text-lyra-muted'
}

export default function StatusState({ kind, title, message, actionLabel, onAction, compact = false }: StatusStateProps) {
  const Icon = iconFor(kind)
  const isError = kind === 'error'
  const isLoading = kind === 'loading'

  return (
    <section
      role={isError ? 'alert' : isLoading ? 'status' : undefined}
      aria-live={isError ? 'assertive' : isLoading ? 'polite' : undefined}
      aria-busy={isLoading || undefined}
      data-state-kind={kind}
      className={`rounded border border-dashed border-lyra-border text-center ${compact ? 'px-4 py-6' : 'px-5 py-10'}`}
    >
      <Icon aria-hidden="true" className={`mx-auto h-5 w-5 ${accentFor(kind)} ${isLoading ? 'motion-safe:animate-spin' : ''}`} />
      <p className="mt-3 text-sm font-medium text-lyra-text">{title}</p>
      {message && <p className="mx-auto mt-1 max-w-2xl text-xs leading-5 text-lyra-mutedDim">{message}</p>}
      {actionLabel && onAction && (
        <button
          type="button"
          onClick={onAction}
          className="mt-4 rounded border border-lyra-gold/40 px-3 py-1.5 font-mono text-[11px] tracking-wider text-lyra-goldBright outline-none focus-visible:ring-2 focus-visible:ring-lyra-goldBright/70"
        >
          {actionLabel}
        </button>
      )}
    </section>
  )
}
