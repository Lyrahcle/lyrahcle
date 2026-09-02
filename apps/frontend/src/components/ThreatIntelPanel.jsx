import { useEffect, useState } from 'react'
import { ArrowDown, ArrowUp } from 'lucide-react'
import SeverityBadge from './SeverityBadge.jsx'
import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'
import Skeleton from './Skeleton.jsx'
import { fetchSpaceCVEKeywords } from '../lib/api'
import { useI18n } from '../lib/I18nContext'
import StatusState from './StatusState.tsx'

const SEVERITIES = ['low', 'medium', 'high', 'critical']
const SORT_FIELDS = ['published', 'severity', 'score', 'cve_id']

function FilterBar({ filters, onFiltersChange, keywords, t }) {
  function update(patch) {
    onFiltersChange({ ...filters, ...patch })
  }

  return (
    <div className="mb-4 flex flex-wrap items-end gap-3">
      <div>
        <label htmlFor="cve-severity" className="mb-1 block text-[10px] tracking-wider text-lyra-mutedDim">
          {t('threatIntel.severity')}
        </label>
        <select
          id="cve-severity"
          value={filters.severity}
          onChange={(e) => update({ severity: e.target.value })}
          className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
        >
          <option value="">{t('threatIntel.allSeverities')}</option>
          {SEVERITIES.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label htmlFor="cve-keyword" className="mb-1 block text-[10px] tracking-wider text-lyra-mutedDim">
          {t('threatIntel.keyword')}
        </label>
        <select
          id="cve-keyword"
          value={filters.keyword}
          onChange={(e) => update({ keyword: e.target.value })}
          className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
        >
          <option value="">{t('threatIntel.allKeywords')}</option>
          {keywords.map((k) => (
            <option key={k} value={k}>
              {k}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label htmlFor="cve-from" className="mb-1 block text-[10px] tracking-wider text-lyra-mutedDim">
          {t('threatIntel.publishedFrom')}
        </label>
        <input
          id="cve-from"
          type="date"
          value={filters.publishedFrom}
          onChange={(e) => update({ publishedFrom: e.target.value })}
          className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
        />
      </div>

      <div>
        <label htmlFor="cve-to" className="mb-1 block text-[10px] tracking-wider text-lyra-mutedDim">
          {t('threatIntel.publishedTo')}
        </label>
        <input
          id="cve-to"
          type="date"
          value={filters.publishedTo}
          onChange={(e) => update({ publishedTo: e.target.value })}
          className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
        />
      </div>

      <div>
        <label htmlFor="cve-sort" className="mb-1 block text-[10px] tracking-wider text-lyra-mutedDim">
          {t('threatIntel.sortBy')}
        </label>
        <div className="flex gap-1.5">
          <select
            id="cve-sort"
            value={filters.sort}
            onChange={(e) => update({ sort: e.target.value })}
            className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
          >
            {SORT_FIELDS.map((f) => (
              <option key={f} value={f}>
                {t(`threatIntel.sortField.${f}`)}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => update({ order: filters.order === 'asc' ? 'desc' : 'asc' })}
            aria-label={filters.order === 'asc' ? t('threatIntel.ascending') : t('threatIntel.descending')}
            title={filters.order === 'asc' ? t('threatIntel.ascending') : t('threatIntel.descending')}
            className="flex items-center justify-center rounded border border-lyra-border px-2 py-1.5 text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
          >
            {filters.order === 'asc' ? <ArrowUp className="h-3.5 w-3.5" strokeWidth={1.75} /> : <ArrowDown className="h-3.5 w-3.5" strokeWidth={1.75} />}
          </button>
        </div>
      </div>

      {(filters.severity || filters.keyword || filters.publishedFrom || filters.publishedTo) && (
        <button
          type="button"
          onClick={() => onFiltersChange({ severity: '', keyword: '', publishedFrom: '', publishedTo: '', sort: filters.sort, order: filters.order })}
          className="rounded border border-lyra-border px-3 py-1.5 font-mono text-[11px] tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
        >
          {t('threatIntel.clearFilters')}
        </button>
      )}
    </div>
  )
}

export default function ThreatIntelPanel({ page, loading, filters, onFiltersChange, onPrev, onNext }) {
  const { t } = useI18n()
  const [keywords, setKeywords] = useState([])

  useEffect(() => {
    fetchSpaceCVEKeywords().then(setKeywords).catch(() => {})
  }, [])

  const cves = page?.items ?? []

  return (
    <div>
      <FilterBar filters={filters} onFiltersChange={onFiltersChange} keywords={keywords} t={t} />

      {loading ? (
        <Skeleton />
      ) : !cves.length ? (
        <StatusState kind="empty" title={t('threatIntel.empty')} message={t('threatIntel.emptyHint')} />
      ) : (
        <div>
          <TableScroll>
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-lyra-borderSoft">
                  <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                    {t('threatIntel.cve')}
                  </th>
                  <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                    {t('threatIntel.severity')}
                  </th>
                  <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                    {t('threatIntel.score')}
                  </th>
                  <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                    {t('threatIntel.published')}
                  </th>
                  <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                    {t('threatIntel.description')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {cves.map((c) => (
                  <tr key={c.cve_id} className="border-b border-lyra-borderSoft/60 last:border-0">
                    <td className="py-3 pr-4 font-mono text-xs text-lyra-goldBright">
                      <a
                        href={`https://nvd.nist.gov/vuln/detail/${c.cve_id}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="hover:underline"
                      >
                        {c.cve_id}
                      </a>
                    </td>
                    <td className="py-3 pr-4"><SeverityBadge severity={c.severity} /></td>
                    <td className="py-3 pr-4 font-mono text-lyra-text">
                      {c.score ? c.score.toFixed(1) : '—'}
                    </td>
                    <td className="py-3 pr-4 font-mono text-xs text-lyra-mutedDim">
                      {c.published ? new Date(c.published).toLocaleDateString() : '—'}
                    </td>
                    <td className="max-w-md py-3 text-xs text-lyra-muted">{c.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
          <Pagination total={page.total} limit={page.limit} offset={page.offset} onPrev={onPrev} onNext={onNext} />
        </div>
      )}
    </div>
  )
}
