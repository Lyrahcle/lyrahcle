import { Search, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { globalSearch } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'

const TYPES = ['all', 'assets', 'satellites', 'cves', 'incidents']

export default function SearchBar({ onNavigate }) {
  const { t } = useI18n()
  const [value, setValue] = useState('')
  const [open, setOpen] = useState(false)
  const [type, setType] = useState('all')
  const [results, setResults] = useState([])
  const [loading, setLoading] = useState(false)
  const containerRef = useRef(null)
  const inputRef = useRef(null)

  useEffect(() => {
    function handleKey(e) {
      if (e.key === '/' && document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
        e.preventDefault()
        inputRef.current?.focus()
      }
      if (e.key === 'Escape') setOpen(false)
    }
    function handleClickOutside(e) {
      if (containerRef.current && !containerRef.current.contains(e.target)) setOpen(false)
    }
    document.addEventListener('keydown', handleKey)
    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('keydown', handleKey)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [])

  useEffect(() => {
    const query = value.trim()
    if (query.length < 2) {
      setResults([])
      setLoading(false)
      return undefined
    }
    const controller = new AbortController()
    const timer = setTimeout(async () => {
      setLoading(true)
      try {
        const items = await globalSearch(query, controller.signal)
        if (!controller.signal.aborted) setResults(items)
      } catch {
        if (!controller.signal.aborted) setResults([])
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    }, 180)
    return () => {
      controller.abort()
      clearTimeout(timer)
    }
  }, [value])

  const visibleResults = type === 'all' ? results : results.filter((item) => item.type === type)

  function handleSelect(result) {
    onNavigate?.(result.tab)
    setOpen(false)
  }

  return (
    <div className="relative hidden md:block" ref={containerRef}>
      <label className="group flex items-center gap-2 rounded border border-lyra-border bg-lyra-panel/40 px-3 py-1.5 transition-colors focus-within:border-lyra-gold/50">
        <Search className="h-3.5 w-3.5 text-lyra-mutedDim group-focus-within:text-lyra-gold" strokeWidth={1.75} />
        <input
          ref={inputRef}
          value={value}
          onChange={(e) => {
            setValue(e.target.value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          type="search"
          autoComplete="off"
          placeholder={t('header.searchPlaceholder')}
          className="w-56 bg-transparent font-mono text-[11px] text-lyra-text placeholder:text-lyra-mutedDim focus:outline-none"
        />
        {value ? (
          <button type="button" onClick={() => { setValue(''); setOpen(false) }} aria-label={t('common.close')} className="text-lyra-mutedDim hover:text-lyra-goldBright">
            <X className="h-3.5 w-3.5" strokeWidth={1.75} />
          </button>
        ) : (
          <kbd className="rounded border border-lyra-borderSoft px-1.5 py-0.5 font-mono text-[10px] text-lyra-mutedDim">/</kbd>
        )}
      </label>

      {open && value.trim().length >= 2 && (
        <div className="absolute left-0 top-full z-50 mt-2 w-96 overflow-hidden rounded border border-lyra-border bg-lyra-panel shadow-lg">
          <div className="border-b border-lyra-borderSoft px-3 py-2">
            <select value={type} onChange={(e) => setType(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-1.5 py-1 font-mono text-[10px] uppercase tracking-wider text-lyra-muted outline-none focus:border-lyra-gold">
              {TYPES.map((item) => <option key={item} value={item}>{t(`header.searchType.${item}`)}</option>)}
            </select>
          </div>
          {loading ? (
            <p className="px-3 py-3 text-xs text-lyra-muted">{t('common.loading')}</p>
          ) : visibleResults.length === 0 ? (
            <p className="px-3 py-3 text-xs text-lyra-muted">{t('header.searchNoResults')}</p>
          ) : (
            <ul className="max-h-80 overflow-y-auto">
              {visibleResults.map((result) => (
                <li key={`${result.type}-${result.id}`}>
                  <button type="button" onClick={() => handleSelect(result)} className="flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left outline-none transition-colors hover:bg-lyra-bgSoft focus-visible:bg-lyra-bgSoft">
                    <span className="font-mono text-[10px] uppercase tracking-wider text-lyra-mutedDim">{t(`header.searchType.${result.type}`)}</span>
                    <span className="max-w-full truncate text-xs text-lyra-text">{result.label}</span>
                    <span className="max-w-full truncate text-[11px] text-lyra-mutedDim">{result.subtitle}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
