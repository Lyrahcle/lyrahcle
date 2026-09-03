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
const TITLE_LEAVES = new Set(['title', 'panelTitle'])
const TITLE_SMALL_WORDS = new Set(['a', 'an', 'and', 'as', 'at', 'but', 'by', 'for', 'in', 'of', 'on', 'or', 'the', 'to', 'via'])

export function englishTitleCase(value: string): string {
  const words = value.split(/(\s+)/)
  const wordIndexes = words.map((word, index) => (/\S/.test(word) ? index : -1)).filter((index) => index >= 0)
  const first = wordIndexes[0]
  const last = wordIndexes[wordIndexes.length - 1]
  return words.map((word, index) => {
    if (!/\S/.test(word) || /^[A-Z0-9/&+.-]+$/.test(word) || /[{}`]/.test(word)) return word
    const lower = word.toLowerCase()
    if (index !== first && index !== last && TITLE_SMALL_WORDS.has(lower)) return lower
    return lower.replace(/^([^a-z0-9]*)([a-z])/, (_, prefix, letter) => `${prefix}${letter.toUpperCase()}`)
  }).join('')
}

function isTitleKey(key: string): boolean {
  return TITLE_LEAVES.has(key.split('.').at(-1) ?? '')
}

function sentenceCaseRomanceTitle(value: string): string {
  const words = value.split(/(\s+)/)
  let seenWord = false
  return words.map((word) => {
    if (!/\S/.test(word)) return word
    const firstWord = !seenWord
    seenWord = true
    if (/^[A-Z0-9/&+.-]+$/.test(word) || /[{}`]/.test(word)) return word
    const lower = word.toLocaleLowerCase()
    return firstWord
      ? lower.replace(/^([^\p{L}\p{N}]*)(\p{L})/u, (_, prefix, letter) => `${prefix}${letter.toLocaleUpperCase()}`)
      : lower
  }).join('')
}

export function formatTitleByLanguage(lang: Lang, value: string): string {
  if (lang === 'en') return englishTitleCase(value)
  if (lang === 'es' || lang === 'fr') return sentenceCaseRomanceTitle(value)
  return value
}

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
	  let template = (lookup(translations[lang], key) ?? lookup(translations[DEFAULT_LANG], key) ?? key) as string
	  if (isTitleKey(key)) template = formatTitleByLanguage(lang, template)
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
