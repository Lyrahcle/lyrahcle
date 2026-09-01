import { useMemo, useState } from 'react'
import KindBadge from './KindBadge.jsx'
import CopyButton from './CopyButton.jsx'
import ConfirmDialog from './ConfirmDialog.jsx'
import TableScroll from './TableScroll.jsx'
import Pagination from './Pagination.jsx'
import { deleteAsset, updateAsset } from '../lib/api'
import { hasPermission } from '../lib/permissions'
import { useI18n } from '../lib/I18nContext'

const KINDS = ['domain', 'ip', 'satellite', 'ground_station', 'firmware']

export default function AssetsTable({ page, programs = [], onPrev, onNext, onChanged }) {
  const { t } = useI18n()
  const assets = page.items
  const canManage = hasPermission('assets.manage')
  const programNames = useMemo(() => new Map(programs.map((p) => [p.id, p.name])), [programs])
  const [editingId, setEditingId] = useState(null)
  const [draft, setDraft] = useState({ programId: '', kind: 'domain', value: '' })
  const [busyId, setBusyId] = useState(null)
  const [error, setError] = useState(null)
  const [pendingDelete, setPendingDelete] = useState(null)

  function beginEdit(asset) {
    setEditingId(asset.id)
    setDraft({ programId: asset.program_id, kind: asset.kind, value: asset.value })
    setError(null)
  }

  async function save(assetId) {
    const value = draft.value.trim()
    if (!draft.programId || !value) {
      setError(t('assetInventory.assetFieldsRequired'))
      return
    }
    setBusyId(assetId)
    setError(null)
    try {
      await updateAsset(assetId, { programId: draft.programId, kind: draft.kind, value })
      setEditingId(null)
      await onChanged?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusyId(null)
    }
  }

  async function confirmRemove() {
    const asset = pendingDelete
    if (!asset) return
    setBusyId(asset.id)
    setError(null)
    try {
      await deleteAsset(asset.id)
      if (editingId === asset.id) setEditingId(null)
      setPendingDelete(null)
      await onChanged?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusyId(null)
    }
  }


  if (!assets.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">{t('assetInventory.noAssets')}</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">{t('assetInventory.noAssetsHint')}</p>
      </div>
    )
  }

  return (
    <div>
      {error && <p role="alert" className="mb-3 text-sm text-lyra-bad">{error}</p>}
      <TableScroll>
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-lyra-borderSoft">
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('assetInventory.kind')}</th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('assetInventory.value')}</th>
              <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('assetInventory.program')}</th>
              {canManage && <th scope="col" className="pb-2 text-right text-[11px] font-normal tracking-wider text-lyra-mutedDim">{t('assetInventory.actions')}</th>}
            </tr>
          </thead>
          <tbody>
            {assets.map((asset) => {
              const editing = editingId === asset.id
              return (
                <tr key={asset.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-3 pr-3">
                    {editing ? (
                      <select
                        aria-label={t('assetInventory.kind')}
                        value={draft.kind}
                        onChange={(e) => setDraft((current) => ({ ...current, kind: e.target.value }))}
                        className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1 text-sm text-lyra-text"
                      >
                        {KINDS.map((kind) => <option key={kind} value={kind}>{kind}</option>)}
                      </select>
                    ) : <KindBadge kind={asset.kind} />}
                  </td>
                  <td className="py-3 pr-3 font-mono text-lyra-text">
                    {editing ? (
                      <input
                        aria-label={t('assetInventory.value')}
                        value={draft.value}
                        onChange={(e) => setDraft((current) => ({ ...current, value: e.target.value }))}
                        className="w-full min-w-48 rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1 text-sm text-lyra-text"
                      />
                    ) : (
                      <div className="flex items-center gap-1">
                        <span className="break-all">{asset.value}</span>
                        <CopyButton value={asset.value} label={t('assetInventory.value')} />
                      </div>
                    )}
                  </td>
                  <td className="py-3 pr-3 text-xs text-lyra-mutedDim">
                    {editing && programs.length > 0 ? (
                      <select
                        aria-label={t('assetInventory.program')}
                        value={draft.programId}
                        onChange={(e) => setDraft((current) => ({ ...current, programId: e.target.value }))}
                        className="max-w-64 rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1 text-sm text-lyra-text"
                      >
                        {programs.map((program) => <option key={program.id} value={program.id}>{program.name}</option>)}
                      </select>
                    ) : (programNames.get(asset.program_id) ?? asset.program_id)}
                  </td>
                  {canManage && (
                    <td className="py-3 text-right">
                      <div className="flex justify-end gap-2">
                        {editing ? (
                          <>
                            <button type="button" disabled={busyId === asset.id} onClick={() => save(asset.id)} className="rounded bg-lyra-gold px-2.5 py-1 text-xs font-medium text-lyra-bg disabled:opacity-50">
                              {busyId === asset.id ? t('common.saving') : t('common.save')}
                            </button>
                            <button type="button" disabled={busyId === asset.id} onClick={() => setEditingId(null)} className="rounded border border-lyra-border px-2.5 py-1 text-xs text-lyra-muted hover:text-lyra-text disabled:opacity-50">{t('common.cancel')}</button>
                          </>
                        ) : (
                          <>
                            <button type="button" onClick={() => beginEdit(asset)} className="rounded border border-lyra-border px-2.5 py-1 text-xs text-lyra-muted hover:border-lyra-gold/40 hover:text-lyra-goldBright">{t('assetInventory.edit')}</button>
                            <button type="button" disabled={busyId === asset.id} onClick={() => setPendingDelete(asset)} className="rounded border border-lyra-bad/40 px-2.5 py-1 text-xs text-lyra-bad hover:bg-lyra-bad/10 disabled:opacity-50">{busyId === asset.id ? t('assetInventory.deleting') : t('assetInventory.delete')}</button>
                          </>
                        )}
                      </div>
                    </td>
                  )}
                </tr>
              )
            })}
          </tbody>
        </table>
      </TableScroll>
      <Pagination total={page.total} limit={page.limit} offset={page.offset} onPrev={onPrev} onNext={onNext} />
      {pendingDelete && (
        <ConfirmDialog
          message={t('assetInventory.deleteAssetConfirm', { value: pendingDelete.value })}
          confirmLabel={t('assetInventory.delete')}
          busy={busyId === pendingDelete.id}
          onConfirm={confirmRemove}
          onCancel={() => setPendingDelete(null)}
        />
      )}
    </div>
  )
}
