import { Menu, Settings } from 'lucide-react'
import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'
import AccountMenu from './AccountMenu.jsx'
import CommandPalette from './CommandPalette.jsx'
import SystemStatusField from './SystemStatusField.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

export default function Header({
  onLogout,
  onHome,
  onOpenSettings,
  onOpenNavigation,
  activePageLabel,
  backendStatus,
  dbStatus,
  sdrStatus,
  navigationItems,
  onCommandNavigate,
}) {
  const { t } = useI18n()

  return (
    <header className="sticky top-0 z-40 border-b border-lyra-borderSoft bg-lyra-bg/85 backdrop-blur">
      <div className="flex items-center justify-between gap-3 px-4 py-3.5 sm:px-6 xl:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <button
            type="button"
            onClick={onOpenNavigation}
            aria-label={t('navigation.open')}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded border border-lyra-border bg-lyra-panel/70 text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold lg:hidden"
          >
            <Menu className="h-4 w-4" strokeWidth={1.8} />
          </button>
          <button
            type="button"
            onClick={onHome}
            className="flex min-w-0 items-center gap-2 rounded outline-none focus-visible:ring-1 focus-visible:ring-lyra-gold lg:hidden"
          >
            <LogoMark className="h-8 w-8 shrink-0" />
            <span className="hidden font-display text-base font-medium tracking-tight text-lyra-goldBright sm:inline">Lyrahcle</span>
          </button>
          <div className="hidden min-w-0 lg:block">
            <p className="font-mono text-[9px] tracking-[0.12em] text-lyra-mutedDim">{t('navigation.workspace')}</p>
            <p className="truncate text-sm font-medium text-lyra-text">{activePageLabel}</p>
          </div>
        </div>

        <div className="flex min-w-0 items-center gap-2 sm:gap-3">
          <SystemStatusField backendStatus={backendStatus} dbStatus={dbStatus} sdrStatus={sdrStatus} />
          <CommandPalette navigationItems={navigationItems} onNavigate={onCommandNavigate} />
          {onOpenSettings && (
            <button
              type="button"
              onClick={onOpenSettings}
              aria-label={t('settings.title')}
              title={t('settings.title')}
              className="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-lyra-border text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
            >
              <Settings className="h-3.5 w-3.5" strokeWidth={1.75} />
            </button>
          )}
          {onLogout && <AccountMenu onLogout={onLogout} />}
          <div className="hidden border-l border-lyra-borderSoft pl-3 xl:block">
            <Clock />
          </div>
        </div>
      </div>
    </header>
  )
}
