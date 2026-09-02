import { useI18n } from '../lib/I18nContext'

export default function SkipLink() {
  const { t } = useI18n()
  return (
    <a
      href="#main-content"
      className="fixed left-4 top-4 z-[130] -translate-y-24 rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg shadow-lg transition-transform focus:translate-y-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-lyra-text"
    >
      {t('ux.skipToContent')}
    </a>
  )
}
