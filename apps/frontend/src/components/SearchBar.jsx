import { Search, X } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useI18n } from '../lib/I18nContext.jsx'

const TYPES = ['all', 'assets', 'satellites', 'cves']
const SORTS = ['relevance', 'az', 'za']

function scoreMatch(haystack, query) {
  const idx = haystack.toLowerCase().indexOf(query)
  if (idx === -1) return -1
  return idx === 0 ? 0 : 1 // prefix matches rank above mid-string matches
}

export default function SearchBar({ assets, satellites, cves, onNavigate }) {
  const { t } = useI18n()
  const [value, setValue] = useState('')
  const [open, setOpen] = useState(false)
  const [type, setType] = useState('all')
  const [sort, setSort] = useState('relevance')
  const containerRef = useRef(null)

  useEffect(() => {
    if (!open) return
    function handleClickOutside(e) {
      if (containerRef.current && !containerRef.current.contains(e.target)) setOpen(false)
    }
    function handleEscape(e) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [open])

  const results = useMemo(() => {
    const query = value.trim().toLowerCase()
    if (!query) return []

    const out = []

    if (type === 'all' || type === 'assets') {
      for (const a of assets?.items ?? []) {
        const label = `${a.kind} · ${a.value}`
        const score = scoreMatch(label, query)
        if (score >= 0) out.push({ id: `asset-${a.id}`, category: 'assets', label, tab: 'assets', score })
      }
    }
    if (type === 'all' || type === 'satellites') {
      for (const s of satellites ?? []) {
        const label = `${s.name} · NORAD ${s.norad_id}`
        const score = scoreMatch(label, query)
        if (score >= 0) out.push({ id: `sat-${s.norad_id}`, category: 'satellites', label, tab: 'orbital', score })
      }
    }
    if (type === 'all' || type === 'cves') {
      for (const c of cves?.items ?? []) {
        const label = `${c.cve_id} · ${c.description ?? ''}`
        const score = scoreMatch(label, query)
        if (score >= 0) out.push({ id: `cve-${c.cve_id}`, category: 'cves', label, tab: 'intel', score })
      }
    }

    if (sort === 'az') out.sort((a, b) => a.label.localeCompare(b.label))
    else if (sort === 'za') out.sort((a, b) => b.label.localeCompare(a.label))
    else out.sort((a, b) => a.score - b.score || a.label.localeCompare(b.label))

    return out.slice(0, 8)
  }, [value, type, sort, assets, satellites, cves])

  function handleSelect(result) {
    onNavigate?.(result.tab)
    setOpen(false)
  }

  return (
    <div className="relative hidden md:block" ref={containerRef}>
      <label className="group flex items-center gap-2 rounded border border-lyra-border bg-lyra-panel/40 px-3 py-1.5 transition-colors focus-within:border-lyra-gold/50">
        <Search className="h-3.5 w-3.5 text-lyra-mutedDim group-focus-within:text-lyra-gold" strokeWidth={1.75} />
        <input
          value={value}
          onChange={(e) => {
            setValue(e.target.value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          type="text"
          placeholder={t('header.searchPlaceholder')}
          className="w-56 bg-transparent font-mono text-[11px] text-lyra-text placeholder:text-lyra-mutedDim focus:outline-none"
        />
        {value ? (
          <button
            type="button"
            onClick={() => {
              setValue('')
              setOpen(false)
            }}
            aria-label={t('common.close')}
            className="text-lyra-mutedDim outline-none transition-colors hover:text-lyra-goldBright"
          >
            <X className="h-3.5 w-3.5" strokeWidth={1.75} />
          </button>
        ) : (
          <kbd className="rounded border border-lyra-borderSoft px-1.5 py-0.5 font-mono text-[10px] text-lyra-mutedDim">
            /
          </kbd>
        )}
      </label>

      {open && value.trim() && (
        <div className="absolute left-0 top-full z-50 mt-2 w-80 overflow-hidden rounded border border-lyra-border bg-lyra-panel shadow-lg">
          <div className="flex items-center gap-2 border-b border-lyra-borderSoft px-3 py-2">
            <select
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="rounded border border-lyra-border bg-lyra-bgSoft px-1.5 py-1 font-mono text-[10px] uppercase tracking-wider text-lyra-muted outline-none focus:border-lyra-gold"
            >
              {TYPES.map((tp) => (
                <option key={tp} value={tp}>
                  {t(`header.searchType.${tp}`)}
                </option>
              ))}
            </select>
            <select
              value={sort}
              onChange={(e) => setSort(e.target.value)}
              className="rounded border border-lyra-border bg-lyra-bgSoft px-1.5 py-1 font-mono text-[10px] uppercase tracking-wider text-lyra-muted outline-none focus:border-lyra-gold"
            >
              {SORTS.map((sr) => (
                <option key={sr} value={sr}>
                  {t(`header.searchSort.${sr}`)}
                </option>
              ))}
            </select>
          </div>

          {results.length === 0 ? (
            <p className="px-3 py-3 text-xs text-lyra-muted">{t('header.searchNoResults')}</p>
          ) : (
            <ul className="max-h-72 overflow-y-auto">
              {results.map((r) => (
                <li key={r.id}>
                  <button
                    type="button"
                    onClick={() => handleSelect(r)}
                    className="flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left outline-none transition-colors hover:bg-lyra-bgSoft focus-visible:bg-lyra-bgSoft"
                  >
                    <span className="font-mono text-[10px] uppercase tracking-wider text-lyra-mutedDim">
                      {t(`header.searchType.${r.category}`)}
                    </span>
                    <span className="truncate text-xs text-lyra-text">{r.label}</span>
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
