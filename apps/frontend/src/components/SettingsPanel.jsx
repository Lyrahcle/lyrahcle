import { useEffect, useState } from 'react'
import { Moon, Sun, Bell, BellOff, Wifi } from 'lucide-react'
import { useI18n } from '../lib/I18nContext.jsx'
import { supportedLanguages } from '../lib/translations.js'
import { getPreferredTheme, setTheme } from '../lib/theme.js'
import { getNotificationsEnabled, isNotificationSupported, setNotificationsEnabled } from '../lib/notifications.js'
import { fetchSystemInfo } from '../lib/api.js'
import ChangePasswordModal from './ChangePasswordModal.jsx'
import Skeleton from './Skeleton.jsx'
import pkg from '../../package.json'

const LANGUAGE_NAMES = { en: 'English', es: 'Español', fr: 'Français', de: 'Deutsch' }

function SettingsCard({ heading, hint, children, className = '' }) {
  return (
    <div className={`rounded-lg border border-lyra-border bg-lyra-panel/30 p-5 sm:p-6 ${className}`}>
      <h3 className="font-display text-sm font-medium text-lyra-text">{heading}</h3>
      <p className="mt-1 text-xs text-lyra-mutedDim">{hint}</p>
      <div className="mt-4">{children}</div>
    </div>
  )
}

function ThemeOption({ value, current, onSelect, label, Icon }) {
  const active = value === current
  return (
    <button
      type="button"
      onClick={() => onSelect(value)}
      aria-pressed={active}
      className={`flex flex-1 items-center justify-center gap-2 rounded border px-4 py-3 text-sm outline-none transition-colors focus-visible:ring-1 focus-visible:ring-lyra-gold ${
        active
          ? 'border-lyra-gold/60 bg-lyra-gold/10 text-lyra-goldBright'
          : 'border-lyra-border text-lyra-muted hover:border-lyra-gold/30 hover:text-lyra-text'
      }`}
    >
      <Icon className="h-4 w-4" strokeWidth={1.75} />
      {label}
    </button>
  )
}

function InfoRow({ label, value }) {
  return (
    <div className="flex items-center justify-between gap-3 py-1.5 text-sm">
      <span className="text-lyra-mutedDim">{label}</span>
      <span className="font-mono text-xs text-lyra-text">{value}</span>
    </div>
  )
}

export default function SettingsPanel({ admin, onNavigateToConnectivity }) {
  const { t, lang, setLang } = useI18n()
  const [theme, setThemeState] = useState(getPreferredTheme)
  const [changingPassword, setChangingPassword] = useState(false)

  const [notifEnabled, setNotifEnabled] = useState(getNotificationsEnabled)
  const [notifError, setNotifError] = useState(null)

  const [systemInfo, setSystemInfo] = useState(null)
  const [systemInfoError, setSystemInfoError] = useState(null)

  useEffect(() => {
    setTheme(theme)
  }, [theme])

  useEffect(() => {
    let cancelled = false
    fetchSystemInfo()
      .then((info) => {
        if (!cancelled) setSystemInfo(info)
      })
      .catch((err) => {
        if (!cancelled) setSystemInfoError(err.message)
      })
    return () => {
      cancelled = true
    }
  }, [])

  async function handleToggleNotifications() {
    setNotifError(null)
    const next = !notifEnabled
    const granted = await setNotificationsEnabled(next)
    setNotifEnabled(granted)
    if (next && !granted) {
      setNotifError(
        isNotificationSupported() ? t('settings.notificationsDenied') : t('settings.notificationsUnsupported'),
      )
    }
  }

  return (
    <div className="grid gap-6 sm:grid-cols-2">
      <SettingsCard heading={t('settings.appearance')} hint={t('settings.appearanceHint')}>
        <div role="group" aria-label={t('settings.appearance')} className="flex gap-3">
          <ThemeOption value="dark" current={theme} onSelect={setThemeState} label={t('settings.dark')} Icon={Moon} />
          <ThemeOption value="light" current={theme} onSelect={setThemeState} label={t('settings.light')} Icon={Sun} />
        </div>
      </SettingsCard>

      <SettingsCard heading={t('settings.language')} hint={t('settings.languageHint')}>
        <div role="group" aria-label={t('settings.language')} className="grid grid-cols-2 gap-3">
          {supportedLanguages.map((code) => (
            <button
              key={code}
              type="button"
              onClick={() => setLang(code)}
              aria-pressed={lang === code}
              className={`flex-1 rounded border px-4 py-3 text-sm outline-none transition-colors focus-visible:ring-1 focus-visible:ring-lyra-gold ${
                lang === code
                  ? 'border-lyra-gold/60 bg-lyra-gold/10 text-lyra-goldBright'
                  : 'border-lyra-border text-lyra-muted hover:border-lyra-gold/30 hover:text-lyra-text'
              }`}
            >
              {LANGUAGE_NAMES[code] ?? code}
            </button>
          ))}
        </div>
      </SettingsCard>

      <SettingsCard heading={t('settings.notifications')} hint={t('settings.notificationsHint')}>
        <button
          type="button"
          onClick={handleToggleNotifications}
          aria-pressed={notifEnabled}
          className={`flex w-full items-center justify-center gap-2 rounded border px-4 py-3 text-sm outline-none transition-colors focus-visible:ring-1 focus-visible:ring-lyra-gold ${
            notifEnabled
              ? 'border-lyra-gold/60 bg-lyra-gold/10 text-lyra-goldBright'
              : 'border-lyra-border text-lyra-muted hover:border-lyra-gold/30 hover:text-lyra-text'
          }`}
        >
          {notifEnabled ? <Bell className="h-4 w-4" strokeWidth={1.75} /> : <BellOff className="h-4 w-4" strokeWidth={1.75} />}
          {notifEnabled ? t('settings.notificationsOn') : t('settings.notificationsOff')}
        </button>
        {notifError && <p className="mt-2 text-xs text-lyra-bad">{notifError}</p>}
      </SettingsCard>

      <SettingsCard heading={t('settings.account')} hint={t('settings.accountHint')}>
        <button
          type="button"
          onClick={() => setChangingPassword(true)}
          className="rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
        >
          {t('settings.changePassword')}
        </button>
      </SettingsCard>

      <SettingsCard heading={t('settings.externalSources')} hint={t('settings.externalSourcesHint')} className="sm:col-span-2">
        {systemInfoError ? (
          <p className="text-sm text-lyra-bad">{systemInfoError}</p>
        ) : systemInfo === null ? (
          <Skeleton rows={2} />
        ) : (
          <div className="grid gap-x-8 gap-y-1 sm:grid-cols-2">
            <InfoRow label={t('settings.celestrakGroups')} value={systemInfo.celestrak_groups?.join(', ')} />
            <InfoRow label={t('settings.celestrakSchedule')} value={systemInfo.celestrak_sync_schedule} />
            <InfoRow label={t('settings.nvdSync')} value={systemInfo.nvd_sync_note} />
          </div>
        )}
        <p className="mt-3 text-xs text-lyra-mutedDim">{t('settings.externalSourcesReadOnlyNote')}</p>
      </SettingsCard>

      <SettingsCard heading={t('settings.security')} hint={t('settings.securityHint')} className="sm:col-span-2">
        <p className="text-sm text-lyra-muted">{t('settings.passwordPolicy')}</p>

        {admin && (
          <div className="mt-5">
            <p className="mb-2 font-mono text-[11px] uppercase tracking-wider text-lyra-gold">
              {t('panels.sdrAgents')}
            </p>
            <p className="mb-3 text-xs text-lyra-mutedDim">{t('settings.agentKeysHint')}</p>
            <button
              type="button"
              onClick={onNavigateToConnectivity}
              className="flex items-center gap-2 rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
            >
              <Wifi className="h-4 w-4" strokeWidth={1.75} />
              {t('settings.manageAgentKeys')}
            </button>
          </div>
        )}
      </SettingsCard>

      <SettingsCard heading={t('settings.systemInfo')} hint={t('settings.systemInfoHint')} className="sm:col-span-2">
        <div className="grid gap-x-8 gap-y-1 sm:grid-cols-2">
          <InfoRow label={t('settings.appVersion')} value={`v${pkg.version}`} />
          <InfoRow label={t('settings.environment')} value={systemInfo?.environment ?? '—'} />
          <InfoRow label={t('settings.license')} value={systemInfo?.license ?? 'AGPLv3 (core)'} />
        </div>
      </SettingsCard>

      {changingPassword && <ChangePasswordModal onClose={() => setChangingPassword(false)} />}
    </div>
  )
}
