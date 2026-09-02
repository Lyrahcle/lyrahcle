import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'
import { Search } from 'lucide-react'
import type { CorrelationNode } from '../../lib/correlationGraph'
import { useI18n } from '../../lib/I18nContext'

interface NodeSearchProps {
  nodes: CorrelationNode[]
  degrees: Map<string, number>
  toneFor: (node: CorrelationNode) => string
  kindLabel: (node: CorrelationNode) => string
  onSelect: (id: string) => void
}

export default function NodeSearch({ nodes, degrees, toneFor, kindLabel, onSelect }: NodeSearchProps) {
  const { t } = useI18n()
  const [query, setQuery] = useState('')
  const [highlight, setHighlight] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const matches = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase()
    if (!needle) return []
    return nodes
      .filter((node) => `${node.label} ${node.subtitle || ''} ${node.kind}`.toLocaleLowerCase().includes(needle))
      .sort((a, b) => {
        const aStarts = a.label.toLocaleLowerCase().startsWith(needle) ? 1 : 0
        const bStarts = b.label.toLocaleLowerCase().startsWith(needle) ? 1 : 0
        return bStarts - aStarts || (degrees.get(b.id) ?? 0) - (degrees.get(a.id) ?? 0) || a.label.localeCompare(b.label)
      })
      .slice(0, 7)
  }, [query, nodes, degrees])

  useEffect(() => {
    setHighlight(0)
  }, [query])

  function commit(id: string) {
    onSelect(id)
    setQuery('')
    inputRef.current?.blur()
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (!matches.length) return
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setHighlight((value) => (value + 1) % matches.length)
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      setHighlight((value) => (value - 1 + matches.length) % matches.length)
    } else if (event.key === 'Enter') {
      event.preventDefault()
      commit(matches[highlight].id)
    } else if (event.key === 'Escape') {
      setQuery('')
    }
  }

  const open = Boolean(query.trim())

  return (
    <div className="relative w-full max-w-sm">
      <label className="relative block">
        <span className="sr-only">{t('correlationGraph.search')}</span>
        <Search className="pointer-events-none absolute left-3 top-2.5 h-3.5 w-3.5 text-lyra-mutedDim" aria-hidden="true" />
        <input
          ref={inputRef}
          type="search"
          role="combobox"
          aria-expanded={open}
          aria-controls="correlation-search-results"
          aria-autocomplete="list"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={t('correlationGraph.search')}
          className="w-full rounded-full border border-lyra-borderSoft/80 bg-[#080a0d]/82 py-2 pl-9 pr-3 text-[11px] text-lyra-text shadow-lg shadow-black/40 backdrop-blur-md placeholder:text-lyra-mutedDim"
        />
      </label>
      {open && (
        <div
          id="correlation-search-results"
          role="listbox"
          aria-label={t('correlationGraph.searchResults')}
          className="absolute bottom-[42px] left-0 right-0 grid max-h-60 gap-0.5 overflow-y-auto rounded-2xl border border-lyra-borderSoft/80 bg-[#080a0d]/95 p-1.5 shadow-2xl shadow-black/50 backdrop-blur-md"
        >
          {matches.length ? (
            matches.map((node, index) => (
              <button
                key={node.id}
                type="button"
                role="option"
                aria-selected={index === highlight}
                onMouseEnter={() => setHighlight(index)}
                onClick={() => commit(node.id)}
                className={`flex items-center gap-2 rounded-xl px-2.5 py-2 text-left transition ${
                  index === highlight ? 'bg-lyra-gold/12 text-lyra-text' : 'text-lyra-muted hover:bg-lyra-gold/8'
                }`}
              >
                <span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ backgroundColor: toneFor(node) }} />
                <span className="min-w-0 flex-1 truncate text-[11px]">{node.label}</span>
                <span className="shrink-0 font-mono text-[8px] tracking-wider text-lyra-mutedDim">
                  {kindLabel(node)}
                </span>
              </button>
            ))
          ) : (
            <span className="px-2.5 py-2 text-[10px] text-lyra-mutedDim">{t('correlationGraph.noSearchResults')}</span>
          )}
        </div>
      )}
    </div>
  )
}
