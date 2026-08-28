import { useState } from 'react'
import { createSatellite } from '../lib/api.js'

export default function NewSatelliteForm({ onCreated }) {
  const [open, setOpen] = useState(false)
  const [noradId, setNoradId] = useState('')
  const [name, setName] = useState('')
  const [line1, setLine1] = useState('')
  const [line2, setLine2] = useState('')
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await createSatellite({
        noradId: Number(noradId),
        name: name.trim(),
        tleLine1: line1.trim(),
        tleLine2: line2.trim(),
      })
      setNoradId('')
      setName('')
      setLine1('')
      setLine2('')
      setOpen(false)
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
        className="rounded border border-lyra-border px-3 py-1.5 font-mono text-xs uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
      >
        + Register satellite
      </button>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="mb-6 rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <label className="mb-1 block text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            NORAD ID
          </label>
          <input
            value={noradId}
            onChange={(e) => setNoradId(e.target.value)}
            type="number"
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-sm text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>
        <div>
          <label className="mb-1 block text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            Name
          </label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="ISS (ZARYA)"
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 text-sm text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>
        <div className="sm:col-span-2">
          <label className="mb-1 block text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            TLE line 1
          </label>
          <input
            value={line1}
            onChange={(e) => setLine1(e.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>
        <div className="sm:col-span-2">
          <label className="mb-1 block text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            TLE line 2
          </label>
          <input
            value={line2}
            onChange={(e) => setLine2(e.target.value)}
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>
      </div>

      {error && <p className="mt-3 text-sm text-lyra-bad">{error}</p>}

      <div className="mt-4 flex gap-2">
        <button
          type="submit"
          disabled={loading}
          className="rounded bg-lyra-gold px-4 py-1.5 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {loading ? 'Registering…' : 'Register'}
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
