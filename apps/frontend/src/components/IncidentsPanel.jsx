import { useEffect, useMemo, useState } from 'react'
import Pagination from './Pagination.jsx'
import SeverityBadge from './SeverityBadge.jsx'
import TableScroll from './TableScroll.jsx'
import { addIncidentEvidence, addIncidentNote, createIncident, fetchIncident, updateIncident } from '../lib/api.js'
import { hasPermission } from '../lib/permissions.js'
import { useI18n } from '../lib/I18nContext.jsx'

const STATUSES = ['new', 'investigating', 'contained', 'resolved', 'false_positive']
const SEVERITIES = ['critical', 'high', 'medium', 'low', 'info']
const EVIDENCE_TYPES = ['rf_anomaly', 'request', 'cve', 'agent', 'satellite', 'external']
const NEXT_STATUSES = {
  new: ['new', 'investigating', 'false_positive'],
  investigating: ['investigating', 'contained', 'resolved', 'false_positive'],
  contained: ['contained', 'investigating', 'resolved'],
  resolved: ['resolved'],
  false_positive: ['false_positive'],
}

export default function IncidentsPanel({ page, filters, onFiltersChange, onPrev, onNext, onChanged, satellites = [], users = [] }) {
  const { t } = useI18n()
  const [selectedId, setSelectedId] = useState(null)
  const [detail, setDetail] = useState(null)
  const [error, setError] = useState(null)
  const [saving, setSaving] = useState(false)
  const [note, setNote] = useState('')
  const [evidence, setEvidence] = useState({ type: 'request', reference: '', label: '' })
  const [form, setForm] = useState({ title: '', description: '', severity: 'medium', satelliteId: '', agentId: '', assignedTo: '' })
  const incidents = page?.items ?? []
  const canManage = hasPermission('incidents.manage')
  const canManageUsers = hasPermission('users.manage')

  useEffect(() => {
    if (!selectedId) {
      setDetail(null)
      return
    }
    let cancelled = false
    fetchIncident(selectedId)
      .then((data) => {
        if (!cancelled) setDetail(data)
      })
      .catch((err) => {
        if (!cancelled) setError(err.message)
      })
    return () => {
      cancelled = true
    }
  }, [selectedId, page])

  const selected = detail?.incident
  const orderedUsers = useMemo(() => [...users].sort((a, b) => (a.username || a.email).localeCompare(b.username || b.email)), [users])

  async function handleCreate(event) {
    event.preventDefault()
    setError(null)
    setSaving(true)
    try {
      await createIncident({
        title: form.title,
        description: form.description,
        severity: form.severity,
        satelliteId: form.satelliteId,
        agentId: form.agentId,
        assignedTo: canManageUsers ? form.assignedTo : '',
      })
      setForm({ title: '', description: '', severity: 'medium', satelliteId: '', agentId: '', assignedTo: '' })
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function applyUpdate(fields) {
    if (!selected) return
    setError(null)
    setSaving(true)
    try {
      await updateIncident(selected.id, fields)
      const next = await fetchIncident(selected.id)
      setDetail(next)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function handleNote(event) {
    event.preventDefault()
    if (!selected || !note.trim()) return
    setError(null)
    setSaving(true)
    try {
      await addIncidentNote(selected.id, note.trim())
      setNote('')
      const next = await fetchIncident(selected.id)
      setDetail(next)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function handleEvidence(event) {
    event.preventDefault()
    if (!selected || !evidence.reference.trim()) return
    setError(null)
    setSaving(true)
    try {
      await addIncidentEvidence(selected.id, { ...evidence, reference: evidence.reference.trim(), label: evidence.label.trim() })
      setEvidence({ type: 'request', reference: '', label: '' })
      const next = await fetchIncident(selected.id)
      setDetail(next)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="grid gap-6">
      {error && <p role="alert" className="text-sm text-lyra-bad">{error}</p>}

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <label className="grid gap-1 text-xs text-lyra-muted">
          {t('incidents.status')}
          <select value={filters.status} onChange={(e) => onFiltersChange({ ...filters, status: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
            <option value="">{t('incidents.allStatuses')}</option>
            {STATUSES.map((status) => <option key={status} value={status}>{t(`incidents.statuses.${status}`)}</option>)}
          </select>
        </label>
        <label className="grid gap-1 text-xs text-lyra-muted">
          {t('incidents.severity')}
          <select value={filters.severity} onChange={(e) => onFiltersChange({ ...filters, severity: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
            <option value="">{t('incidents.allSeverities')}</option>
            {SEVERITIES.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
          </select>
        </label>
        <label className="grid gap-1 text-xs text-lyra-muted">
          {t('incidents.agent')}
          <input value={filters.agentId} onChange={(e) => onFiltersChange({ ...filters, agentId: e.target.value })} maxLength={128} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" placeholder="sdr-agent-01" />
        </label>
        <label className="grid gap-1 text-xs text-lyra-muted">
          {t('incidents.satellite')}
          <select value={filters.satelliteId} onChange={(e) => onFiltersChange({ ...filters, satelliteId: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
            <option value="">{t('incidents.allSatellites')}</option>
            {satellites.map((satellite) => <option key={satellite.id} value={satellite.id}>{satellite.name}</option>)}
          </select>
        </label>
      </div>

      {canManage && (
        <form onSubmit={handleCreate} className="grid gap-3 rounded border border-lyra-borderSoft bg-lyra-bgSoft/40 p-4">
          <div className="grid gap-3 md:grid-cols-3">
            <input aria-label={t('incidents.title')} required maxLength={160} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} placeholder={t('incidents.titlePlaceholder')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text md:col-span-2" />
            <select aria-label={t('incidents.severity')} value={form.severity} onChange={(e) => setForm({ ...form, severity: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
              {SEVERITIES.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
            </select>
          </div>
          <textarea aria-label={t('incidents.descriptionPlaceholder')} maxLength={4000} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder={t('incidents.descriptionPlaceholder')} className="min-h-20 rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
          <div className="grid gap-3 md:grid-cols-3">
            <select aria-label={t('incidents.noSatellite')} value={form.satelliteId} onChange={(e) => setForm({ ...form, satelliteId: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
              <option value="">{t('incidents.noSatellite')}</option>
              {satellites.map((satellite) => <option key={satellite.id} value={satellite.id}>{satellite.name}</option>)}
            </select>
            <input aria-label={t('incidents.agent')} maxLength={128} value={form.agentId} onChange={(e) => setForm({ ...form, agentId: e.target.value })} placeholder={t('incidents.agentPlaceholder')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
            {canManageUsers ? (
              <select aria-label={t('incidents.unassigned')} value={form.assignedTo} onChange={(e) => setForm({ ...form, assignedTo: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
                <option value="">{t('incidents.unassigned')}</option>
                {orderedUsers.map((user) => <option key={user.id} value={user.id}>{user.username || user.email}</option>)}
              </select>
            ) : <span />}
          </div>
          <button disabled={saving} className="justify-self-start rounded border border-lyra-gold/40 px-4 py-2 font-mono text-xs uppercase tracking-wide text-lyra-goldBright disabled:opacity-50">
            {saving ? t('common.saving') : t('incidents.create')}
          </button>
        </form>
      )}

      {!incidents.length ? (
        <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center text-sm text-lyra-muted">{t('incidents.empty')}</div>
      ) : (
        <>
          <TableScroll>
            <table className="w-full text-left text-sm">
              <thead><tr className="border-b border-lyra-borderSoft">
                {[t('incidents.reference'), t('incidents.title'), t('incidents.severity'), t('incidents.status'), t('incidents.source'), t('incidents.updated')].map((label) => <th key={label} className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{label}</th>)}
              </tr></thead>
              <tbody>
                {incidents.map((incident) => (
                  <tr key={incident.id} className="border-b border-lyra-borderSoft/60 last:border-0 hover:bg-lyra-borderSoft/30">
                    <td className="py-3 pr-4 font-mono text-xs text-lyra-goldBright"><button type="button" onClick={() => setSelectedId(incident.id)} className="outline-none hover:underline focus:underline">{incident.reference}</button></td>
                    <td className="py-3 pr-4 text-xs text-lyra-text">{incident.title}</td>
                    <td className="py-3 pr-4"><SeverityBadge severity={incident.severity} /></td>
                    <td className="py-3 pr-4 font-mono text-[11px] uppercase text-lyra-muted">{t(`incidents.statuses.${incident.status}`)}</td>
                    <td className="py-3 pr-4 font-mono text-xs text-lyra-mutedDim">{incident.source_type}</td>
                    <td className="py-3 font-mono text-xs text-lyra-mutedDim">{new Date(incident.updated_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
          <Pagination total={page.total} limit={page.limit} offset={page.offset} onPrev={onPrev} onNext={onNext} />
        </>
      )}

      {selected && (
        <section className="grid gap-4 rounded border border-lyra-border p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="font-mono text-xs text-lyra-goldBright">{selected.reference}</p>
              <h3 className="mt-1 text-lg text-lyra-text">{selected.title}</h3>
              {selected.description && <p className="mt-2 max-w-3xl text-sm text-lyra-muted">{selected.description}</p>}
            </div>
            <SeverityBadge severity={selected.severity} />
          </div>

          {canManage && (
            <div className="grid gap-3 sm:grid-cols-3">
              <select aria-label={t('incidents.status')} value={selected.status} disabled={saving} onChange={(e) => applyUpdate({ status: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
                {(NEXT_STATUSES[selected.status] || [selected.status]).map((status) => <option key={status} value={status}>{t(`incidents.statuses.${status}`)}</option>)}
              </select>
              <select aria-label={t('incidents.severity')} value={selected.severity} disabled={saving} onChange={(e) => applyUpdate({ severity: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
                {SEVERITIES.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
              </select>
              {canManageUsers && (
                <select aria-label={t('incidents.unassigned')} value={selected.assigned_to || ''} disabled={saving} onChange={(e) => applyUpdate({ assignedTo: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
                  <option value="">{t('incidents.unassigned')}</option>
                  {orderedUsers.map((user) => <option key={user.id} value={user.id}>{user.username || user.email}</option>)}
                </select>
              )}
            </div>
          )}

          <div>
            <h4 className="mb-2 font-mono text-xs uppercase tracking-wider text-lyra-muted">{t('incidents.timeline')}</h4>
            <div className="grid gap-2">
              {(detail.events || []).map((event) => (
                <div key={event.id} className="rounded border border-lyra-borderSoft px-3 py-2">
                  <div className="flex flex-wrap justify-between gap-2">
                    <span className="font-mono text-[11px] uppercase text-lyra-goldBright">{event.event_type}</span>
                    <span className="font-mono text-[11px] text-lyra-mutedDim">{new Date(event.created_at).toLocaleString()}</span>
                  </div>
                  {event.actor_username && <p className="mt-1 text-xs text-lyra-muted">{event.actor_username}</p>}
                  {event.message && <p className="mt-1 text-sm text-lyra-text">{event.message}</p>}
                </div>
              ))}
            </div>
          </div>

          <div>
            <h4 className="mb-2 font-mono text-xs uppercase tracking-wider text-lyra-muted">{t('incidents.evidence')}</h4>
            <div className="grid gap-2">
              {(detail.evidence || []).map((item) => (
                <div key={item.id} className="rounded border border-lyra-borderSoft px-3 py-2">
                  <div className="flex flex-wrap justify-between gap-2">
                    <span className="font-mono text-[11px] uppercase text-lyra-goldBright">{item.evidence_type}</span>
                    <span className="font-mono text-[11px] text-lyra-mutedDim">{new Date(item.created_at).toLocaleString()}</span>
                  </div>
                  <p className="mt-1 break-all font-mono text-xs text-lyra-text">{item.evidence_ref}</p>
                  {item.label && <p className="mt-1 text-xs text-lyra-muted">{item.label}</p>}
                </div>
              ))}
            </div>
          </div>

          {canManage && (
            <form onSubmit={handleEvidence} className="grid gap-2 sm:grid-cols-[160px_1fr_1fr_auto]">
              <select value={evidence.type} onChange={(e) => setEvidence({ ...evidence, type: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
                {EVIDENCE_TYPES.map((type) => <option key={type} value={type}>{type}</option>)}
              </select>
              <input required maxLength={256} value={evidence.reference} onChange={(e) => setEvidence({ ...evidence, reference: e.target.value })} placeholder={t('incidents.evidenceReference')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
              <input maxLength={160} value={evidence.label} onChange={(e) => setEvidence({ ...evidence, label: e.target.value })} placeholder={t('incidents.evidenceLabel')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
              <button disabled={saving || !evidence.reference.trim()} className="rounded border border-lyra-border px-3 py-2 font-mono text-xs uppercase text-lyra-goldBright disabled:opacity-50">{t('incidents.addEvidence')}</button>
            </form>
          )}

          {canManage && (
            <form onSubmit={handleNote} className="flex gap-2">
              <input aria-label={t('incidents.notePlaceholder')} value={note} onChange={(e) => setNote(e.target.value)} maxLength={2000} placeholder={t('incidents.notePlaceholder')} className="min-w-0 flex-1 rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
              <button disabled={saving || !note.trim()} className="rounded border border-lyra-border px-3 py-2 font-mono text-xs uppercase text-lyra-goldBright disabled:opacity-50">{t('incidents.addNote')}</button>
            </form>
          )}
        </section>
      )}
    </div>
  )
}
