import { useEffect, useState } from 'react'
import { Moon, Sun } from 'lucide-react'
import { useI18n } from '../lib/I18nContext.jsx'
import { supportedLanguages } from '../lib/translations.js'
import { getPreferredTheme, setTheme } from '../lib/theme.js'
import ChangePasswordModal from './ChangePasswordModal.jsx'

const LANGUAGE_NAMES = { en: 'English', es: 'Español' }

function SettingsCard({ heading, hint, children }) {
  return (
    <div className="rounded-lg border border-lyra-border bg-lyra-panel/30 p-5 sm:p-6">
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

export default function SettingsPanel() {
  const { t, lang, setLang } = useI18n()
  const [theme, setThemeState] = useState(getPreferredTheme)
  const [changingPassword, setChangingPassword] = useState(false)

  useEffect(() => {
    setTheme(theme)
  }, [theme])

  return (
    <div className="grid gap-6 sm:grid-cols-2">
      <SettingsCard heading={t('settings.appearance')} hint={t('settings.appearanceHint')}>
        <div role="group" aria-label={t('settings.appearance')} className="flex gap-3">
          <ThemeOption value="dark" current={theme} onSelect={setThemeState} label={t('settings.dark')} Icon={Moon} />
          <ThemeOption value="light" current={theme} onSelect={setThemeState} label={t('settings.light')} Icon={Sun} />
        </div>
      </SettingsCard>

      <SettingsCard heading={t('settings.language')} hint={t('settings.languageHint')}>
        <div role="group" aria-label={t('settings.language')} className="flex gap-3">
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

      <SettingsCard heading={t('settings.account')} hint={t('settings.accountHint')}>
        <button
          type="button"
          onClick={() => setChangingPassword(true)}
          className="rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
        >
          {t('settings.changePassword')}
        </button>
      </SettingsCard>

      {changingPassword && <ChangePasswordModal onClose={() => setChangingPassword(false)} />}
    </div>
  )
}
