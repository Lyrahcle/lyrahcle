import { Fragment, useState } from 'react'
import { Pencil, Save, X } from 'lucide-react'
import TableScroll from './TableScroll.jsx'
import { updateSatellite } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'

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

export default function SatellitesPanel({ satellites, canManage = false, onChanged = async () => {} }) {
  const { t } = useI18n()
  const [editingId, setEditingId] = useState(null)

  if (!satellites.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">{t('satelliteManagement.empty')}</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">{t('satelliteManagement.emptyHint')}</p>
      </div>
    )
  }

  return (
    <TableScroll>
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-lyra-borderSoft">
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('satelliteManagement.satellite')}</th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('satelliteForm.noradId')}</th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('satelliteManagement.latitude')}</th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('satelliteManagement.longitude')}</th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('satelliteManagement.altitude')}</th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('satelliteManagement.tleStatus')}</th>
          </tr>
        </thead>
        <tbody>
          {satellites.map((s) => (
            <Fragment key={s.id}>
              <tr className="border-b border-lyra-borderSoft/60 last:border-0">
                <td className="py-3 text-lyra-goldBright">
                  <div className="flex items-center gap-2">
                    <span>{s.name}</span>
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
                <td className="py-3 font-mono text-xs text-lyra-mutedDim">{s.norad_id}</td>
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
