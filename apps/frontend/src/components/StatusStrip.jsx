import StatusPill from './StatusPill.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

const STATUS_LABELS = {
  en: { online: 'online', offline: 'offline', checking: 'checking', live: 'live', idle: 'idle' },
  es: { online: 'en línea', offline: 'sin conexión', checking: 'comprobando', live: 'en vivo', idle: 'inactivo' },
}

export default function StatusStrip({ backendStatus, dbStatus, sdrStatus }) {
  const { t, lang } = useI18n()
  const words = STATUS_LABELS[lang] ?? STATUS_LABELS.en
  const label = (status) => words[status] ?? status

  return (
    <div className="border-b border-lyra-borderSoft bg-lyra-bgSoft/40">
      <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-2 px-6 py-2.5 sm:px-8">
        <StatusPill label={`${t('status.backend')} · ${label(backendStatus)}`} tone={toneFor(backendStatus)} />
        <StatusPill label={`${t('status.database')} · ${label(dbStatus)}`} tone={toneFor(dbStatus)} />
        <StatusPill label={`${t('status.sdrAgent')} · ${label(sdrStatus)}`} tone={toneFor(sdrStatus)} />
      </div>
    </div>
  )
}

function toneFor(status) {
  if (status === 'online' || status === 'live') return 'good'
  if (status === 'idle' || status === 'checking') return 'warn'
  if (status === 'offline') return 'bad'
  return 'idle'
}
