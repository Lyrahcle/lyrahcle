import { useEffect, useState } from 'react'
import { createAsset, createProgram, fetchPrograms } from '../lib/api'

const KINDS = ['domain', 'ip', 'satellite', 'ground_station', 'firmware']

export default function NewAssetForm({ onCreated, programs: providedPrograms = null, onProgramsChanged }) {
  const [open, setOpen] = useState(false)
  const [programs, setPrograms] = useState(providedPrograms)
  const [programId, setProgramId] = useState('')
  const [newProgramName, setNewProgramName] = useState('')
  const [kind, setKind] = useState('domain')
  const [value, setValue] = useState('')
  const [cpeUri, setCpeUri] = useState('')
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (providedPrograms !== null) {
      setPrograms(providedPrograms)
      if (!programId && providedPrograms.length) setProgramId(providedPrograms[0].id)
      return
    }
    if (!open || programs !== null) return
    fetchPrograms()
      .then((data) => {
        setPrograms(data ?? [])
        if (data?.length) setProgramId(data[0].id)
      })
      .catch(() => setPrograms([]))
  }, [open, programs, providedPrograms, programId])

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      let targetProgramId = programId
      if (!targetProgramId && newProgramName.trim()) {
        const program = await createProgram(newProgramName.trim())
        targetProgramId = program.id
        await onProgramsChanged?.()
      }
      if (!targetProgramId) {
        throw new Error('Select an existing program or name a new one')
      }
      await createAsset({ programId: targetProgramId, kind, value: value.trim(), cpeUri: cpeUri.trim() })
      setValue('')
      setCpeUri('')
      setNewProgramName('')
      onCreated()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="rounded border border-lyra-border px-3 py-1.5 font-mono text-xs tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
      >
        + New asset
      </button>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="mb-6 rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <label htmlFor="asset-program" className="mb-1 block text-[11px] tracking-wider text-lyra-mutedDim">
            Program
          </label>
          <select
            id="asset-program"
            value={programId}
            onChange={(e) => setProgramId(e.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 text-sm text-lyra-text outline-none focus:border-lyra-gold"
          >
            <option value="">— new program below —</option>
            {(programs ?? []).map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          {!programId && (
            <>
              <label htmlFor="asset-new-program" className="sr-only">
                New program name
              </label>
              <input
                id="asset-new-program"
                value={newProgramName}
                onChange={(e) => setNewProgramName(e.target.value)}
                placeholder="New program name"
                className="mt-1.5 w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 text-sm text-lyra-text outline-none focus:border-lyra-gold"
              />
            </>
          )}
        </div>

        <div>
          <label htmlFor="asset-kind" className="mb-1 block text-[11px] tracking-wider text-lyra-mutedDim">
            Kind
          </label>
          <select
            id="asset-kind"
            value={kind}
            onChange={(e) => setKind(e.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 text-sm text-lyra-text outline-none focus:border-lyra-gold"
          >
            {KINDS.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </select>
        </div>

        <div className="sm:col-span-2">
          <label htmlFor="asset-value" className="mb-1 block text-[11px] tracking-wider text-lyra-mutedDim">
            Value
          </label>
          <input
            id="asset-value"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="e.g. ops.example.com, 198.51.100.42, NORAD 25544"
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-sm text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>

        <div className="sm:col-span-2">
          <label htmlFor="asset-cpe" className="mb-1 block text-[11px] tracking-wider text-lyra-mutedDim">
            CPE 2.3 (optional)
          </label>
          <input
            id="asset-cpe"
            value={cpeUri}
            onChange={(e) => setCpeUri(e.target.value)}
            placeholder="cpe:2.3:a:vendor:product:version:*:*:*:*:*:*:*"
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
          />
          <p className="mt-1 text-[10px] text-lyra-mutedDim">Used for product/version applicability matching against NVD configurations.</p>
        </div>
      </div>

      {error && (
        <p role="alert" className="mt-3 text-sm text-lyra-bad">
          {error}
        </p>
      )}

      <div className="mt-4 flex gap-2">
        <button
          type="submit"
          disabled={loading}
          className="rounded bg-lyra-gold px-4 py-1.5 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {loading ? 'Creating…' : 'Create'}
        </button>
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="rounded border border-lyra-border px-4 py-1.5 text-sm text-lyra-muted outline-none hover:text-lyra-text"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}
