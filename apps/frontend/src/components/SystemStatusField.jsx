import { useI18n } from '../lib/I18nContext.jsx'

const STATUS_LABELS = {
  en: { online: 'online', offline: 'offline', checking: 'checking', live: 'live', idle: 'idle' },
  es: { online: 'en línea', offline: 'sin conexión', checking: 'comprobando', live: 'en vivo', idle: 'inactivo' },
}

const TONES = {
  good: 'bg-lyra-good',
  warn: 'bg-lyra-warn',
  bad: 'bg-lyra-bad',
  idle: 'bg-lyra-mutedDim',
}

function toneFor(status) {
  if (status === 'online' || status === 'live') return 'good'
  if (status === 'idle' || status === 'checking') return 'warn'
  if (status === 'offline') return 'bad'
  return 'idle'
}

function Dot({ shortLabel, fullLabel, status }) {
  const tone = toneFor(status)
  return (
    <span className="flex items-center gap-1.5" title={fullLabel}>
      <span className={`h-1.5 w-1.5 rounded-full ${TONES[tone]}`} />
      <span className="font-mono text-[10px] uppercase tracking-wider text-lyra-mutedDim">{shortLabel}</span>
    </span>
  )
}

export default function SystemStatusField({ backendStatus, dbStatus, sdrStatus }) {
  const { t, lang } = useI18n()
  const words = STATUS_LABELS[lang] ?? STATUS_LABELS.en
  const label = (status) => words[status] ?? status

  return (
    <div
      role="status"
      aria-live="polite"
      className="hidden items-center gap-3 rounded border border-lyra-border bg-lyra-panel/40 px-3 py-1.5 lg:flex"
    >
      <Dot shortLabel={t('status.backendShort')} fullLabel={`${t('status.backend')} · ${label(backendStatus)}`} status={backendStatus} />
      <Dot shortLabel={t('status.databaseShort')} fullLabel={`${t('status.database')} · ${label(dbStatus)}`} status={dbStatus} />
      <Dot shortLabel={t('status.sdrAgentShort')} fullLabel={`${t('status.sdrAgent')} · ${label(sdrStatus)}`} status={sdrStatus} />
    </div>
  )
}
