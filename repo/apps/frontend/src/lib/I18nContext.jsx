import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { supportedLanguages, translations } from './translations.js'

const STORAGE_KEY = 'lyra_lang'
const DEFAULT_LANG = 'en'

function detectLanguage() {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (supportedLanguages.includes(stored)) return stored

  const browserLangs = navigator.languages ?? [navigator.language]
  for (const lang of browserLangs) {
    const short = lang?.slice(0, 2).toLowerCase()
    if (supportedLanguages.includes(short)) return short
  }
  return DEFAULT_LANG
}

function lookup(dict, key) {
  return key.split('.').reduce((node, part) => (node && typeof node === 'object' ? node[part] : undefined), dict)
}

const I18nContext = createContext(null)

export function I18nProvider({ children }) {
  const [lang, setLangState] = useState(detectLanguage)

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

  const setLang = useCallback((next) => {
    if (!supportedLanguages.includes(next)) return
    localStorage.setItem(STORAGE_KEY, next)
    setLangState(next)
  }, [])

  const t = useCallback(
    (key, vars) => {
      const template =
        lookup(translations[lang], key) ?? lookup(translations[DEFAULT_LANG], key) ?? key
      if (!vars) return template
      return Object.entries(vars).reduce(
        (str, [name, value]) => str.replaceAll(`{${name}}`, String(value)),
        template,
      )
    },
    [lang],
  )

  const value = useMemo(() => ({ lang, setLang, t }), [lang, setLang, t])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within an I18nProvider')
  return ctx
}
