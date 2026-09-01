import { useState } from 'react'
import SeverityBadge from './SeverityBadge.jsx'
import TableScroll from './TableScroll.jsx'
import { createDetectionRule, updateDetectionRule } from '../lib/api.js'
import { hasPermission } from '../lib/permissions.js'
import { useI18n } from '../lib/I18nContext.jsx'

const METRICS = ['any', 'power_dbm', 'frequency_hz', 'power_dbm_trend', 'frequency_hz_trend']
const OPERATORS = ['abs_z_gte', 'z_gte', 'z_lte']
const SEVERITIES = ['critical', 'high', 'medium', 'low', 'info']

export default function DetectionRulesPanel({ page, onChanged }) {
  const { t } = useI18n()
  const canManage = hasPermission('detections.manage')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState({
    name: '',
    description: '',
    metric: 'any',
    operator: 'abs_z_gte',
    threshold: '6',
    severity: 'high',
    cooldownSeconds: '120',
    autoIncident: true,
  })

  async function handleCreate(event) {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      await createDetectionRule({
        ...form,
        threshold: Number(form.threshold),
        cooldownSeconds: Number(form.cooldownSeconds),
      })
      setForm({ name: '', description: '', metric: 'any', operator: 'abs_z_gte', threshold: '6', severity: 'high', cooldownSeconds: '120', autoIncident: true })
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  async function toggleRule(rule) {
    setError('')
    setSaving(true)
    try {
      await updateDetectionRule(rule.id, { enabled: !rule.enabled })
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  const rules = page?.items ?? []

  return (
    <div className="grid gap-5">
      {error && <p role="alert" className="text-sm text-lyra-bad">{error}</p>}

      {canManage && (
        <form onSubmit={handleCreate} className="grid gap-3 rounded border border-lyra-borderSoft p-4">
          <div className="grid gap-3 md:grid-cols-2">
            <input required maxLength={120} value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={t('detections.namePlaceholder')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
            <input maxLength={1000} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder={t('detections.descriptionPlaceholder')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            <select value={form.metric} onChange={(e) => setForm({ ...form, metric: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
              {METRICS.map((metric) => <option key={metric} value={metric}>{t(`detections.metrics.${metric}`)}</option>)}
            </select>
            <select value={form.operator} onChange={(e) => setForm({ ...form, operator: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
              {OPERATORS.map((operator) => <option key={operator} value={operator}>{t(`detections.operators.${operator}`)}</option>)}
            </select>
            <input type="number" min="0.1" max="100" step="0.1" required value={form.threshold} onChange={(e) => setForm({ ...form, threshold: e.target.value })} aria-label={t('detections.threshold')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
            <select value={form.severity} onChange={(e) => setForm({ ...form, severity: e.target.value })} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text">
              {SEVERITIES.map((severity) => <option key={severity} value={severity}>{severity}</option>)}
            </select>
            <input type="number" min="0" max="86400" required value={form.cooldownSeconds} onChange={(e) => setForm({ ...form, cooldownSeconds: e.target.value })} aria-label={t('detections.cooldown')} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text" />
          </div>
          <label className="flex items-center gap-2 text-xs text-lyra-muted">
            <input type="checkbox" checked={form.autoIncident} onChange={(e) => setForm({ ...form, autoIncident: e.target.checked })} />
            {t('detections.autoIncident')}
          </label>
          <button disabled={saving} className="justify-self-start rounded border border-lyra-gold/40 px-4 py-2 font-mono text-xs tracking-wide text-lyra-goldBright disabled:opacity-50">
            {saving ? t('common.saving') : t('detections.create')}
          </button>
        </form>
      )}

      {!rules.length ? (
        <div className="rounded border border-dashed border-lyra-border px-5 py-8 text-center text-sm text-lyra-muted">{t('detections.empty')}</div>
      ) : (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead><tr className="border-b border-lyra-borderSoft">
              {[t('detections.name'), t('detections.condition'), t('incidents.severity'), t('detections.cooldown'), t('detections.incident'), t('detections.status')].map((label) => <th key={label} className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{label}</th>)}
            </tr></thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-3 pr-4">
                    <p className="text-xs text-lyra-text">{rule.name}</p>
                    {rule.description && <p className="mt-1 max-w-sm text-[11px] text-lyra-mutedDim">{rule.description}</p>}
                  </td>
                  <td className="py-3 pr-4 font-mono text-xs text-lyra-muted">{t(`detections.metrics.${rule.metric}`)} · {t(`detections.operators.${rule.operator}`)} {rule.threshold}</td>
                  <td className="py-3 pr-4"><SeverityBadge severity={rule.severity} /></td>
                  <td className="py-3 pr-4 font-mono text-xs text-lyra-muted">{rule.cooldown_seconds}s</td>
                  <td className="py-3 pr-4 font-mono text-xs text-lyra-muted">{rule.auto_incident ? t('detections.yes') : t('detections.no')}</td>
                  <td className="py-3">
                    {canManage ? (
                      <button type="button" disabled={saving} onClick={() => toggleRule(rule)} className="rounded border border-lyra-border px-2 py-1 font-mono text-[11px] text-lyra-goldBright disabled:opacity-50">
                        {rule.enabled ? t('detections.enabled') : t('detections.disabled')}
                      </button>
                    ) : (
                      <span className="font-mono text-[11px] text-lyra-muted">{rule.enabled ? t('detections.enabled') : t('detections.disabled')}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
    </div>
  )
}
