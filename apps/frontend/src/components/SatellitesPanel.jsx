import { Fragment, useMemo, useState } from 'react'
import { ChevronDown, ChevronUp, ChevronsUpDown, Pencil, Save, Shield, X } from 'lucide-react'
import CopyButton from './CopyButton.jsx'
import TableScroll from './TableScroll.jsx'
import { updateSatellite } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'

const SORT_ACCESSORS = {
  name: (s) => s.name ?? '',
  norad_id: (s) => s.norad_id ?? null,
  latitude_deg: (s) => s.position?.latitude_deg ?? null,
  longitude_deg: (s) => s.position?.longitude_deg ?? null,
  altitude_km: (s) => s.position?.altitude_km ?? null,
  tle_status: (s) => s.tle_status ?? '',
}

function compareValues(a, b) {
  if (typeof a === 'string' && typeof b === 'string') {
    return a.localeCompare(b, undefined, { sensitivity: 'base' })
  }
  return a < b ? -1 : a > b ? 1 : 0
}

function sortSatellites(satellites, sortKey, sortDir) {
  if (!sortKey) return satellites
  const accessor = SORT_ACCESSORS[sortKey]
  return [...satellites].sort((a, b) => {
    const va = accessor(a)
    const vb = accessor(b)
    // Missing values always sort last, independent of ascending/descending.
    if (va == null && vb == null) return 0
    if (va == null) return 1
    if (vb == null) return -1
    const cmp = compareValues(va, vb)
    return sortDir === 'desc' ? -cmp : cmp
  })
}

function SatelliteEditRow({ satellite, onCancel, onSaved }) {
  const { t } = useI18n()
  const [noradId, setNoradId] = useState(String(satellite.norad_id ?? ''))
  const [name, setName] = useState(satellite.name ?? '')
  const [line1, setLine1] = useState(satellite.tle_line1 ?? '')
  const [line2, setLine2] = useState(satellite.tle_line2 ?? '')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  async function submit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await updateSatellite(satellite.id, {
        noradId: Number(noradId),
        name: name.trim(),
        tleLine1: line1.trim(),
        tleLine2: line2.trim(),
      })
      await onSaved()
      onCancel()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <tr className="border-b border-lyra-borderSoft/60 bg-lyra-panel/20">
      <td colSpan={6} className="py-4">
        <form onSubmit={submit} className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/40 p-4">
          <div className="mb-3 flex items-center justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-wider text-lyra-mutedDim">{t('satelliteManagement.editEyebrow')}</p>
              <h4 className="mt-1 text-sm font-medium text-lyra-text">{satellite.name}</h4>
            </div>
            <button type="button" onClick={onCancel} aria-label={t('common.cancel')} className="rounded p-1 text-lyra-muted hover:text-lyra-text">
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <label className="text-xs text-lyra-mutedDim">
              {t('satelliteForm.noradId')}
              <input
                aria-label={t('satelliteForm.noradId')}
                type="number"
                value={noradId}
                onChange={(e) => setNoradId(e.target.value)}
                required
                className="mt-1 w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm text-lyra-text outline-none focus:border-lyra-gold"
              />
            </label>
            <label className="text-xs text-lyra-mutedDim">
              {t('satelliteForm.name')}
              <input
                aria-label={t('satelliteForm.name')}
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="mt-1 w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
              />
            </label>
            <label className="text-xs text-lyra-mutedDim sm:col-span-2">
              {t('satelliteForm.tle1')}
              <textarea
                aria-label={t('satelliteForm.tle1')}
                rows={2}
                value={line1}
                onChange={(e) => setLine1(e.target.value)}
                required
                className="mt-1 w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
              />
            </label>
            <label className="text-xs text-lyra-mutedDim sm:col-span-2">
              {t('satelliteForm.tle2')}
              <textarea
                aria-label={t('satelliteForm.tle2')}
                rows={2}
                value={line2}
                onChange={(e) => setLine2(e.target.value)}
                required
                className="mt-1 w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
              />
            </label>
          </div>

          {error && <p role="alert" className="mt-3 text-sm text-lyra-bad">{error}</p>}

          <div className="mt-4 flex gap-2">
            <button type="submit" disabled={loading} className="flex items-center gap-2 rounded bg-lyra-gold px-3 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">
              <Save className="h-3.5 w-3.5" />
              {loading ? t('common.saving') : t('satelliteManagement.saveChanges')}
            </button>
            <button type="button" onClick={onCancel} className="rounded border border-lyra-border px-3 py-2 text-sm text-lyra-muted hover:text-lyra-text">
              {t('common.cancel')}
            </button>
          </div>
        </form>
      </td>
    </tr>
  )
}

function SortableHeader({ label, sortKey, activeKey, direction, onSort, t, align = 'left' }) {
  const active = sortKey === activeKey
  const ariaSort = active ? (direction === 'desc' ? 'descending' : 'ascending') : 'none'
  const Icon = active ? (direction === 'desc' ? ChevronDown : ChevronUp) : ChevronsUpDown
  return (
    <th scope="col" aria-sort={ariaSort} className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        aria-label={`${t('satelliteManagement.sortBy', { column: label })}${active ? `, ${t(`satelliteManagement.sort${direction === 'desc' ? 'Descending' : 'Ascending'}`)}` : ''}`}
        className={`flex items-center gap-1 rounded uppercase tracking-wider text-lyra-mutedDim hover:text-lyra-text focus-visible:outline focus-visible:outline-1 focus-visible:outline-lyra-gold ${align === 'right' ? 'ml-auto' : ''}`}
      >
        <span>{label}</span>
        <Icon className={`h-3 w-3 ${active ? 'text-lyra-goldBright' : 'text-lyra-mutedDim/60'}`} aria-hidden="true" />
      </button>
    </th>
  )
}

export default function SatellitesPanel({ satellites, canManage = false, onChanged = async () => {}, onOpenProfile = () => {} }) {
  const { t } = useI18n()
  const [editingId, setEditingId] = useState(null)
  const [sortKey, setSortKey] = useState(null)
  const [sortDir, setSortDir] = useState('asc')

  const sortedSatellites = useMemo(
    () => sortSatellites(satellites, sortKey, sortDir),
    [satellites, sortKey, sortDir]
  )

  function handleSort(key) {
    if (sortKey === key) {
      setSortDir((dir) => (dir === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  if (!satellites.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">{t('satelliteManagement.empty')}</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">{t('satelliteManagement.emptyHint')}</p>
      </div>
    )
  }

  const headerProps = { activeKey: sortKey, direction: sortDir, onSort: handleSort, t }

  return (
    <TableScroll>
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-lyra-borderSoft">
            <SortableHeader label={t('satelliteManagement.satellite')} sortKey="name" {...headerProps} />
            <SortableHeader label={t('satelliteForm.noradId')} sortKey="norad_id" {...headerProps} />
            <SortableHeader label={t('satelliteManagement.latitude')} sortKey="latitude_deg" {...headerProps} />
            <SortableHeader label={t('satelliteManagement.longitude')} sortKey="longitude_deg" {...headerProps} />
            <SortableHeader label={t('satelliteManagement.altitude')} sortKey="altitude_km" {...headerProps} />
            <SortableHeader label={t('satelliteManagement.tleStatus')} sortKey="tle_status" {...headerProps} />
          </tr>
        </thead>
        <tbody>
          {sortedSatellites.map((s) => (
            <Fragment key={s.id}>
              <tr className="border-b border-lyra-borderSoft/60 last:border-0">
                <td className="py-3 text-lyra-goldBright">
                  <div className="flex items-center gap-2">
                    <span>{s.name}</span>
                    <button
                      type="button"
                      onClick={() => onOpenProfile(s)}
                      aria-label={t('satelliteProfile.openProfile', { name: s.name })}
                      className="rounded p-1 text-lyra-mutedDim hover:bg-lyra-panel hover:text-lyra-goldBright"
                    >
                      <Shield className="h-3.5 w-3.5" />
                    </button>
                    {canManage && (
                      <button
                        type="button"
                        onClick={() => setEditingId(editingId === s.id ? null : s.id)}
                        aria-label={t('satelliteManagement.editSatellite', { name: s.name })}
                        className="rounded p-1 text-lyra-mutedDim hover:bg-lyra-panel hover:text-lyra-goldBright"
                      >
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                  <p className="mt-1 font-mono text-[10px] text-lyra-mutedDim">
                    {t('satelliteManagement.updated')} {s.updated_at ? new Date(s.updated_at).toLocaleString() : '—'}
                  </p>
                </td>
                <td className="py-3 font-mono text-xs text-lyra-mutedDim">
                  <div className="flex items-center gap-1">
                    <span>{s.norad_id}</span>
                    <CopyButton value={s.norad_id} label={t('satelliteForm.noradId')} />
                  </div>
                </td>
                <td className="py-3 font-mono text-lyra-text">{s.position ? `${s.position.latitude_deg.toFixed(2)}°` : '—'}</td>
                <td className="py-3 font-mono text-lyra-text">{s.position ? `${s.position.longitude_deg.toFixed(2)}°` : '—'}</td>
                <td className="py-3 font-mono text-lyra-text">{s.position ? `${s.position.altitude_km.toFixed(0)} km` : '—'}</td>
                <td className="py-3">
                  <span className="rounded border border-lyra-borderSoft px-2 py-1 font-mono text-[10px] uppercase tracking-wider text-lyra-muted">
                    {s.tle_status ?? 'unknown'}
                  </span>
                  <p className="mt-1 text-[10px] text-lyra-mutedDim">{s.orbital_confidence ?? '—'}</p>
                </td>
              </tr>
              {editingId === s.id && (
                <SatelliteEditRow satellite={s} onCancel={() => setEditingId(null)} onSaved={onChanged} />
              )}
            </Fragment>
          ))}
        </tbody>
      </table>
    </TableScroll>
  )
}
