import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { supportedLanguages, translations } from './translations.js'

export type Lang = string
export type TranslateVars = Record<string, unknown>
export type Translate = (key: string, vars?: TranslateVars) => string

export interface I18nContextValue {
  lang: Lang
  setLang: (next: Lang) => void
  t: Translate
}

const STORAGE_KEY = 'lyra_lang'
const DEFAULT_LANG: Lang = 'en'

function detectLanguage(): Lang {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored && (supportedLanguages as string[]).includes(stored)) return stored

  const browserLangs = navigator.languages ?? [navigator.language]
  for (const lang of browserLangs) {
    const short = lang?.slice(0, 2).toLowerCase()
    if (short && (supportedLanguages as string[]).includes(short)) return short
  }
  return DEFAULT_LANG
}

function lookup(dict: unknown, key: string): unknown {
  return key
    .split('.')
    .reduce<unknown>((node, part) => (node && typeof node === 'object' ? (node as Record<string, unknown>)[part] : undefined), dict)
}

const I18nContext = createContext<I18nContextValue | null>(null)

export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectLanguage)

  useEffect(() => {
    document.documentElement.lang = lang
  }, [lang])

  const setLang = useCallback((next: Lang) => {
    if (!(supportedLanguages as string[]).includes(next)) return
    localStorage.setItem(STORAGE_KEY, next)
    setLangState(next)
  }, [])

  const t = useCallback<Translate>(
    (key, vars) => {
      const template = (lookup(translations[lang], key) ?? lookup(translations[DEFAULT_LANG], key) ?? key) as string
      if (!vars) return template
      return Object.entries(vars).reduce((str, [name, value]) => str.replaceAll(`{${name}}`, String(value)), template)
    },
    [lang],
  )

  const value = useMemo<I18nContextValue>(() => ({ lang, setLang, t }), [lang, setLang, t])

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}

export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within an I18nProvider')
  return ctx
}
