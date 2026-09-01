import { useEffect } from 'react'
import {
  Orbit,
  MapPinned,
  Boxes,
  RadioTower,
  ShieldAlert,
  Radar,
  Binary,
  Cable,
  UsersRound,
  Settings,
  Building2,
  PanelLeftClose,
  PanelLeftOpen,
  X,
} from 'lucide-react'
import LogoMark from './LogoMark.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

const ICONS = {
  orbital: Orbit,
  stations: MapPinned,
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

const SECTION_ORDER = ['overview', 'mission', 'security', 'rf', 'platform', 'preferences']

function groupItems(items) {
  return SECTION_ORDER.map((section) => ({
    section,
    items: items.filter((item) => item.section === section),
  })).filter((group) => group.items.length > 0)
}

function NavigationContent({ items, active, onChange, collapsed, onToggleCollapse, mobile, onCloseMobile }) {
  const { t } = useI18n()
  const groups = groupItems(items)

  function select(id) {
    onChange(id)
    onCloseMobile?.()
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className={`flex h-[65px] shrink-0 items-center border-b border-lyra-borderSoft ${collapsed && !mobile ? 'justify-center px-3' : 'justify-between px-4'}`}>
        <button
          type="button"
          onClick={() => select('orbital')}
          className="flex min-w-0 items-center gap-3 rounded outline-none focus-visible:ring-1 focus-visible:ring-lyra-gold"
          aria-label={t('navigation.home')}
        >
          <LogoMark className="h-9 w-9 shrink-0" />
          {(!collapsed || mobile) && (
            <span className="min-w-0 text-left">
              <span className="block font-display text-lg font-medium tracking-tight text-lyra-goldBright">Lyrahcle</span>
              <span className="block truncate font-mono text-[9px] tracking-[0.16em] text-lyra-mutedDim">{t('header.tagline')}</span>
            </span>
          )}
        </button>
        {mobile ? (
          <button
            type="button"
            onClick={onCloseMobile}
            aria-label={t('navigation.close')}
            className="flex h-8 w-8 items-center justify-center rounded border border-lyra-border text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
          >
            <X className="h-4 w-4" strokeWidth={1.8} />
          </button>
        ) : null}
      </div>

      <nav aria-label={t('navigation.primary')} className="min-h-0 flex-1 overflow-y-auto px-3 py-4">
        <div className="grid gap-5">
          {groups.map((group) => (
            <section key={group.section} aria-label={t(`navigation.sections.${group.section}`)}>
              {!collapsed && (
                <p className="mb-1.5 px-2 font-mono text-[9px] tracking-[0.12em] text-lyra-mutedDim">
                  {t(`navigation.sections.${group.section}`)}
                </p>
              )}
              <div className="grid gap-1">
                {group.items.map((item) => {
                  const Icon = ICONS[item.id] ?? Orbit
                  const selected = item.id === active
                  return (
                    <button
                      key={item.id}
                      type="button"
                      aria-current={selected ? 'page' : undefined}
                      aria-label={collapsed && !mobile ? item.label : undefined}
                      title={collapsed && !mobile ? item.label : undefined}
                      onClick={() => select(item.id)}
                      className={`group flex w-full items-center rounded-lg outline-none transition-colors focus-visible:ring-1 focus-visible:ring-lyra-gold ${
                        collapsed && !mobile ? 'justify-center px-2 py-2.5' : 'gap-3 px-3 py-2.5 text-left'
                      } ${
                        selected
                          ? 'bg-lyra-gold/10 text-lyra-goldBright'
                          : 'text-lyra-muted hover:bg-white/[0.035] hover:text-lyra-text'
                      }`}
                    >
                      <span className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-md border transition-colors ${selected ? 'border-lyra-gold/30 bg-lyra-gold/10' : 'border-lyra-borderSoft bg-black/10 group-hover:border-lyra-border'}`}>
                        <Icon className="h-4 w-4" strokeWidth={1.7} />
                      </span>
                      {(!collapsed || mobile) && (
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm font-medium">{item.label}</span>
                          {item.id === 'orbital' && (
                            <span className="mt-0.5 block truncate text-[10px] text-lyra-mutedDim">{t('navigation.defaultDashboard')}</span>
                          )}
                        </span>
                      )}
                    </button>
                  )
                })}
              </div>
            </section>
          ))}
        </div>
      </nav>

      {!mobile && (
        <div className="shrink-0 border-t border-lyra-borderSoft p-3">
          <button
            type="button"
            onClick={onToggleCollapse}
            aria-label={collapsed ? t('navigation.expand') : t('navigation.collapse')}
            title={collapsed ? t('navigation.expand') : t('navigation.collapse')}
            className={`flex w-full items-center rounded-lg border border-lyra-borderSoft text-lyra-muted outline-none transition-colors hover:border-lyra-gold/35 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold ${collapsed ? 'justify-center p-2.5' : 'gap-3 px-3 py-2.5'}`}
          >
            {collapsed ? <PanelLeftOpen className="h-4 w-4" strokeWidth={1.7} /> : <PanelLeftClose className="h-4 w-4" strokeWidth={1.7} />}
            {!collapsed && <span className="text-xs">{t('navigation.collapse')}</span>}
          </button>
        </div>
      )}
    </div>
  )
}

export default function NavigationMenu({ items, active, onChange, collapsed, onToggleCollapse, mobileOpen, onCloseMobile }) {
  const { t } = useI18n()

  useEffect(() => {
    if (!mobileOpen) return undefined
    function handleKey(event) {
      if (event.key === 'Escape') onCloseMobile()
    }
    document.addEventListener('keydown', handleKey)
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', handleKey)
      document.body.style.overflow = previous
    }
  }, [mobileOpen, onCloseMobile])

  return (
    <>
      <aside
        className={`fixed inset-y-0 left-0 z-50 hidden border-r border-lyra-borderSoft bg-lyra-panel/95 backdrop-blur-xl transition-[width] duration-200 lg:block ${collapsed ? 'w-20' : 'w-64'}`}
      >
        <NavigationContent
          items={items}
          active={active}
          onChange={onChange}
          collapsed={collapsed}
          onToggleCollapse={onToggleCollapse}
        />
      </aside>

      {mobileOpen && (
        <div className="fixed inset-0 z-[80] lg:hidden">
          <button
            type="button"
            aria-label={t('navigation.close')}
            onClick={onCloseMobile}
            className="absolute inset-0 bg-black/65 backdrop-blur-sm"
          />
          <aside className="relative h-full w-[min(19rem,86vw)] border-r border-lyra-border bg-lyra-panel shadow-2xl">
            <NavigationContent
              items={items}
              active={active}
              onChange={onChange}
              collapsed={false}
              mobile
              onCloseMobile={onCloseMobile}
            />
          </aside>
        </div>
      )}
    </>
  )
}
