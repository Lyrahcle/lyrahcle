import { useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import {
  Boxes,
  Command,
  FileWarning,
  Orbit,
  Search,
  ShieldAlert,
  X,
} from 'lucide-react'
import { globalSearch } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import { useFocusTrap } from '../lib/useFocusTrap.js'

const RESULT_ICONS = {
  assets: Boxes,
  satellites: Orbit,
  cves: FileWarning,
  incidents: ShieldAlert,
}

function normalize(value) {
  return String(value ?? '').trim().toLocaleLowerCase()
}

function matchesNavigation(item, query, sectionLabel) {
  if (!query) return true
  const haystack = `${item.label} ${sectionLabel}`.toLocaleLowerCase()
  return haystack.includes(query)
}

export default function CommandPalette({ navigationItems = [], onNavigate }) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState([])
  const [loading, setLoading] = useState(false)
  const [activeIndex, setActiveIndex] = useState(0)
  const dialogRef = useFocusTrap(open)
  const inputRef = useRef(null)

  useEffect(() => {
    function handleShortcut(event) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLocaleLowerCase() === 'k') {
        event.preventDefault()
        setOpen(true)
      }
    }
    document.addEventListener('keydown', handleShortcut)
    return () => document.removeEventListener('keydown', handleShortcut)
  }, [])

  useEffect(() => {
    if (!open) return undefined
    const previous = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    setQuery('')
    setResults([])
    setActiveIndex(0)
    function handleEscape(event) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.body.style.overflow = previous
    }
  }, [open])

  useEffect(() => {
    const value = query.trim()
    if (!open || value.length < 2) {
      setResults([])
      setLoading(false)
      return undefined
    }
    const controller = new AbortController()
    const timer = setTimeout(async () => {
      setLoading(true)
      try {
        const items = await globalSearch(value, controller.signal)
        if (!controller.signal.aborted) setResults(Array.isArray(items) ? items : [])
      } catch {
        if (!controller.signal.aborted) setResults([])
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    }, 160)
    return () => {
      controller.abort()
      clearTimeout(timer)
    }
  }, [open, query])

  const allowedTabs = useMemo(() => new Set(navigationItems.map((item) => item.id)), [navigationItems])
  const normalizedQuery = normalize(query)
  const navigationCommands = useMemo(() => navigationItems
    .filter((item) => matchesNavigation(item, normalizedQuery, t(`navigation.sections.${item.section}`)))
    .map((item) => ({
      commandId: `navigation-${item.id}`,
      kind: 'navigation',
      tab: item.id,
      label: item.label,
      subtitle: t(`navigation.sections.${item.section}`),
    })), [navigationItems, normalizedQuery, t])
  const searchCommands = useMemo(() => results
    .filter((result) => allowedTabs.has(result.tab))
    .map((result) => ({
      ...result,
      commandId: `search-${result.type}-${result.id}`,
      kind: 'search',
    })), [allowedTabs, results])
  const commands = [...navigationCommands, ...searchCommands]

  useEffect(() => {
    setActiveIndex(0)
  }, [query, results.length, navigationCommands.length])

  function close() {
    setOpen(false)
  }

  function select(command) {
    if (!command) return
    onNavigate?.(command.tab, command.kind === 'search' ? command : null)
    close()
  }

  function handleInputKeyDown(event) {
    if (event.key === 'Escape') {
      event.preventDefault()
      close()
      return
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      if (commands.length > 0) setActiveIndex((current) => (current + 1) % commands.length)
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      if (commands.length > 0) setActiveIndex((current) => (current - 1 + commands.length) % commands.length)
      return
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      select(commands[activeIndex])
    }
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={t('commandPalette.open')}
        className="group flex h-8 min-w-8 items-center justify-center gap-2 rounded border border-lyra-border bg-lyra-panel/40 px-2 text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold sm:px-3"
      >
        <Search className="h-3.5 w-3.5" strokeWidth={1.75} />
        <span className="hidden text-[11px] md:inline">{t('commandPalette.trigger')}</span>
        <kbd className="hidden rounded border border-lyra-borderSoft px-1.5 py-0.5 font-mono text-[9px] text-lyra-mutedDim lg:inline">Ctrl K</kbd>
      </button>

      {open && createPortal(
        <div className="fixed inset-0 z-[110] bg-black/70 p-3 backdrop-blur-[2px] sm:p-6" onMouseDown={(event) => { if (event.target === event.currentTarget) close() }}>
          <div className="mx-auto flex min-h-full max-w-2xl items-start justify-center pt-[8vh] sm:pt-[12vh]">
            <div
              ref={dialogRef}
              role="dialog"
              aria-modal="true"
              aria-labelledby="command-palette-title"
              tabIndex={-1}
              className="w-full overflow-hidden rounded-xl border border-lyra-border bg-lyra-panel shadow-2xl shadow-black/50 outline-none"
            >
              <div className="flex items-center gap-3 border-b border-lyra-borderSoft px-4 py-3">
                <Command className="h-4 w-4 shrink-0 text-lyra-gold" strokeWidth={1.7} />
                <div className="min-w-0 flex-1">
                  <h2 id="command-palette-title" className="sr-only">{t('commandPalette.title')}</h2>
                  <input
                    ref={inputRef}
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                    onKeyDown={handleInputKeyDown}
                    type="search"
                    autoComplete="off"
                    aria-label={t('commandPalette.inputLabel')}
                    aria-controls="command-palette-results"
                    aria-activedescendant={commands[activeIndex]?.commandId}
                    placeholder={t('commandPalette.placeholder')}
                    className="w-full bg-transparent text-sm text-lyra-text placeholder:text-lyra-mutedDim focus:outline-none"
                  />
                </div>
                {loading && <span className="font-mono text-[10px] text-lyra-mutedDim">{t('common.loading')}</span>}
                <button
                  type="button"
                  onClick={close}
                  aria-label={t('common.close')}
                  className="flex h-7 w-7 shrink-0 items-center justify-center rounded text-lyra-muted outline-none transition-colors hover:bg-white/[0.04] hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
                >
                  <X className="h-4 w-4" strokeWidth={1.8} />
                </button>
              </div>

              <div id="command-palette-results" role="listbox" aria-label={t('commandPalette.results')} className="max-h-[60vh] overflow-y-auto p-2">
                {navigationCommands.length > 0 && (
                  <section aria-labelledby="command-navigation-heading">
                    <p id="command-navigation-heading" className="px-2 pb-1 pt-1 font-mono text-[9px] tracking-[0.12em] text-lyra-mutedDim">{t('commandPalette.navigation')}</p>
                    <div className="grid gap-1">
                      {navigationCommands.map((command) => {
                        const index = commands.findIndex((item) => item.commandId === command.commandId)
                        const active = index === activeIndex
                        return (
                          <button
                            id={command.commandId}
                            key={command.commandId}
                            type="button"
                            role="option"
                            aria-selected={active}
                            onMouseEnter={() => setActiveIndex(index)}
                            onClick={() => select(command)}
                            className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left outline-none transition-colors ${active ? 'bg-lyra-gold/10 text-lyra-goldBright' : 'text-lyra-text hover:bg-white/[0.035]'}`}
                          >
                            <Command className="h-4 w-4 shrink-0 text-lyra-mutedDim" strokeWidth={1.7} />
                            <span className="min-w-0 flex-1">
                              <span className="block truncate text-sm">{command.label}</span>
                              <span className="mt-0.5 block truncate text-[10px] text-lyra-mutedDim">{command.subtitle}</span>
                            </span>
                            <span className="font-mono text-[9px] text-lyra-mutedDim">{t('commandPalette.jump')}</span>
                          </button>
                        )
                      })}
                    </div>
                  </section>
                )}

                {searchCommands.length > 0 && (
                  <section aria-labelledby="command-search-heading" className={navigationCommands.length > 0 ? 'mt-3 border-t border-lyra-borderSoft pt-2' : ''}>
                    <p id="command-search-heading" className="px-2 pb-1 pt-1 font-mono text-[9px] tracking-[0.12em] text-lyra-mutedDim">{t('commandPalette.searchResults')}</p>
                    <div className="grid gap-1">
                      {searchCommands.map((command) => {
                        const Icon = RESULT_ICONS[command.type] ?? Search
                        const index = commands.findIndex((item) => item.commandId === command.commandId)
                        const active = index === activeIndex
                        return (
                          <button
                            id={command.commandId}
                            key={command.commandId}
                            type="button"
                            role="option"
                            aria-selected={active}
                            onMouseEnter={() => setActiveIndex(index)}
                            onClick={() => select(command)}
                            className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left outline-none transition-colors ${active ? 'bg-lyra-gold/10 text-lyra-goldBright' : 'text-lyra-text hover:bg-white/[0.035]'}`}
                          >
                            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-lyra-borderSoft bg-black/10">
                              <Icon className="h-4 w-4" strokeWidth={1.7} />
                            </span>
                            <span className="min-w-0 flex-1">
                              <span className="block truncate text-sm">{command.label}</span>
                              <span className="mt-0.5 block truncate text-[10px] text-lyra-mutedDim">{t(`header.searchType.${command.type}`)} · {command.subtitle}</span>
                            </span>
                            {command.type === 'satellites' && <span className="font-mono text-[9px] text-lyra-mutedDim">{t('commandPalette.profile')}</span>}
                          </button>
                        )
                      })}
                    </div>
                  </section>
                )}

                {!loading && commands.length === 0 && (
                  <div className="px-4 py-10 text-center">
                    <Search className="mx-auto h-5 w-5 text-lyra-mutedDim" strokeWidth={1.6} />
                    <p className="mt-3 text-sm text-lyra-muted">{normalizedQuery.length < 2 ? t('commandPalette.startTyping') : t('commandPalette.noResults')}</p>
                  </div>
                )}
              </div>

              <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-lyra-borderSoft px-4 py-2 font-mono text-[9px] text-lyra-mutedDim">
                <span>↑ ↓ {t('commandPalette.move')}</span>
                <span>Enter {t('commandPalette.select')}</span>
                <span>Esc {t('commandPalette.close')}</span>
              </div>
            </div>
          </div>
        </div>,
        document.body,
      )}
    </>
  )
}
