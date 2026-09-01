import { useI18n } from '../lib/I18nContext'

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
      <span className="font-mono text-[10px] tracking-wider text-lyra-mutedDim">{shortLabel}</span>
    </span>
  )
}

export default function SystemStatusField({ backendStatus, dbStatus, sdrStatus }) {
  const { t } = useI18n()
  const label = (status) => t(`status.state.${status}`)

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
