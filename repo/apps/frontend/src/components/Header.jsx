import { Settings } from 'lucide-react'
import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'
import AccountMenu from './AccountMenu.jsx'
import SearchBar from './SearchBar.jsx'
import SystemStatusField from './SystemStatusField.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

export default function Header({
  onLogout,
  onHome,
  onOpenSettings,
  backendStatus,
  dbStatus,
  sdrStatus,
  searchAssets,
  searchSatellites,
  searchCves,
  onSearchNavigate,
}) {
  const { t } = useI18n()

  return (
    <header className="sticky top-0 z-40 border-b border-lyra-borderSoft bg-lyra-bg/85 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-3 px-6 py-3.5 sm:px-8">
        <button
          onClick={onHome}
          className="flex items-center gap-3 rounded outline-none focus-visible:ring-1 focus-visible:ring-lyra-gold"
        >
          <LogoMark className="h-9 w-9" />
          <span className="font-display text-lg font-medium tracking-tight text-lyra-goldBright">
            Lyrahcle
          </span>
          <span className="hidden font-mono text-[10px] uppercase tracking-[0.25em] text-lyra-mutedDim sm:inline">
            {t('header.tagline')}
          </span>
        </button>

        <div className="flex items-center gap-3">
          <SystemStatusField backendStatus={backendStatus} dbStatus={dbStatus} sdrStatus={sdrStatus} />

          <SearchBar
            assets={searchAssets}
            satellites={searchSatellites}
            cves={searchCves}
            onNavigate={onSearchNavigate}
          />
          {onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              aria-label={t('settings.title')}
              title={t('settings.title')}
              className="flex h-8 w-8 items-center justify-center rounded border border-lyra-border text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
            >
              <Settings className="h-3.5 w-3.5" strokeWidth={1.75} />
            </button>
          )}

          {onLogout && <AccountMenu onLogout={onLogout} />}

          <div className="hidden border-l border-lyra-borderSoft pl-3 lg:block">
            <Clock />
          </div>
        </div>
      </div>
    </header>
  )
}
