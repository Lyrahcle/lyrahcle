import { useState } from 'react'
import ConfirmDialog from './ConfirmDialog.jsx'
import { deleteProgram, updateProgram } from '../lib/api.js'
import { hasPermission } from '../lib/permissions.js'
import { useI18n } from '../lib/I18nContext.jsx'

export default function ProgramManager({ programs = [], onChanged }) {
  const { t } = useI18n()
  const canManage = hasPermission('programs.manage')
  const [editingId, setEditingId] = useState(null)
  const [name, setName] = useState('')
  const [busyId, setBusyId] = useState(null)
  const [error, setError] = useState(null)
  const [pendingDelete, setPendingDelete] = useState(null)

  function beginEdit(program) {
    setEditingId(program.id)
    setName(program.name)
    setError(null)
  }

  async function save(programId) {
    const trimmed = name.trim()
    if (!trimmed) {
      setError(t('assetInventory.programNameRequired'))
      return
    }
    setBusyId(programId)
    setError(null)
    try {
      await updateProgram(programId, trimmed)
      setEditingId(null)
      await onChanged?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusyId(null)
    }
  }

  async function confirmRemove() {
    const program = pendingDelete
    if (!program) return
    setBusyId(program.id)
    setError(null)
    try {
      await deleteProgram(program.id)
      if (editingId === program.id) setEditingId(null)
      setPendingDelete(null)
      await onChanged?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusyId(null)
    }
  }

  return (
    <section className="mt-6 border-t border-lyra-borderSoft pt-5" aria-labelledby="asset-programs-title">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <h3 id="asset-programs-title" className="text-sm font-medium text-lyra-text">{t('assetInventory.programs')}</h3>
          <p className="mt-1 text-xs text-lyra-mutedDim">{t('assetInventory.programsHint')}</p>
        </div>
      </div>

      {error && <p role="alert" className="mb-3 text-sm text-lyra-bad">{error}</p>}

      {programs.length === 0 ? (
        <p className="text-sm text-lyra-muted">{t('assetInventory.noPrograms')}</p>
      ) : (
        <div className="grid gap-2">
          {programs.map((program) => (
            <div key={program.id} className="flex flex-col gap-2 rounded border border-lyra-borderSoft bg-lyra-bgSoft/20 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
              {editingId === program.id ? (
                <label className="flex min-w-0 flex-1 items-center gap-2 text-xs text-lyra-muted">
                  <span className="shrink-0">{t('assetInventory.programName')}</span>
                  <input
                    aria-label={t('assetInventory.programName')}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="min-w-0 flex-1 rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                  />
                </label>
              ) : (
                <div className="min-w-0">
                  <p className="truncate text-sm text-lyra-text">{program.name}</p>
                  <p className="truncate font-mono text-[10px] text-lyra-mutedDim">{program.id}</p>
                </div>
              )}

              {canManage && (
                <div className="flex shrink-0 gap-2">
                  {editingId === program.id ? (
                    <>
                      <button
                        type="button"
                        disabled={busyId === program.id}
                        onClick={() => save(program.id)}
                        className="rounded bg-lyra-gold px-3 py-1 text-xs font-medium text-lyra-bg disabled:opacity-50"
                      >
                        {busyId === program.id ? t('common.saving') : t('common.save')}
                      </button>
                      <button
                        type="button"
                        disabled={busyId === program.id}
                        onClick={() => setEditingId(null)}
                        className="rounded border border-lyra-border px-3 py-1 text-xs text-lyra-muted hover:text-lyra-text disabled:opacity-50"
                      >
                        {t('common.cancel')}
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        type="button"
                        onClick={() => beginEdit(program)}
                        className="rounded border border-lyra-border px-3 py-1 text-xs text-lyra-muted hover:border-lyra-gold/40 hover:text-lyra-goldBright"
                      >
                        {t('assetInventory.edit')}
                      </button>
                      <button
                        type="button"
                        disabled={busyId === program.id}
                        onClick={() => setPendingDelete(program)}
                        className="rounded border border-lyra-bad/40 px-3 py-1 text-xs text-lyra-bad hover:bg-lyra-bad/10 disabled:opacity-50"
                      >
                        {busyId === program.id ? t('assetInventory.deleting') : t('assetInventory.delete')}
                      </button>
                    </>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
      {pendingDelete && (
        <ConfirmDialog
          message={t('assetInventory.deleteProgramConfirm', { name: pendingDelete.name })}
          confirmLabel={t('assetInventory.delete')}
          busy={busyId === pendingDelete.id}
          onConfirm={confirmRemove}
          onCancel={() => setPendingDelete(null)}
        />
      )}
    </section>
  )
}
