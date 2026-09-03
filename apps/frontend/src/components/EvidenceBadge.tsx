import { useI18n } from '../lib/I18nContext'

export type EvidenceKind = 'live' | 'external' | 'simulated' | 'replayed' | 'inferred' | 'confirmed' | 'context'

const classes: Record<EvidenceKind, string> = {
  live: 'border-lyra-good/35 bg-lyra-good/10 text-lyra-good',
  external: 'border-sky-400/35 bg-sky-400/10 text-sky-300',
  simulated: 'border-fuchsia-400/35 bg-fuchsia-400/10 text-fuchsia-300',
  replayed: 'border-violet-400/35 bg-violet-400/10 text-violet-300',
  inferred: 'border-lyra-gold/35 bg-lyra-gold/10 text-lyra-goldBright',
  confirmed: 'border-lyra-good/35 bg-lyra-good/10 text-lyra-good',
  context: 'border-lyra-warn/35 bg-lyra-warn/10 text-lyra-warn',
}

export default function EvidenceBadge({ kind, label }: { kind: EvidenceKind, label?: string }) {
  const { t } = useI18n()
  return (
    <span className={`inline-flex rounded-full border px-2 py-0.5 font-mono text-[9px] tracking-wider ${classes[kind]}`}>
      {label ?? t(`evidence.kind.${kind}`)}
    </span>
  )
}

