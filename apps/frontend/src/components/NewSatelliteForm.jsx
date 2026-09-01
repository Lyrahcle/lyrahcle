import { useState } from 'react'
import { Rocket } from 'lucide-react'
import { createSatellite } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'

export default function NewSatelliteForm({ onCreated }) {
  const { t } = useI18n()
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
        className="mb-6 flex items-center gap-2 rounded border border-lyra-border px-3 py-1.5 font-mono text-xs tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
      >
        <Rocket className="h-3.5 w-3.5" strokeWidth={1.75} />
        {t('satelliteForm.registerButton')}
      </button>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="mb-6 rounded-lg border border-lyra-border bg-lyra-panel/30 p-5 sm:p-6">
      <div className="mb-4 flex items-center gap-2">
        <Rocket className="h-4 w-4 text-lyra-gold" strokeWidth={1.75} />
        <h3 className="font-display text-sm font-medium text-lyra-text">{t('satelliteForm.title')}</h3>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label htmlFor="sat-norad-id" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">
            {t('satelliteForm.noradId')}
          </label>
          <input
            id="sat-norad-id"
            value={noradId}
            onChange={(e) => setNoradId(e.target.value)}
            type="number"
            placeholder="25544"
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>
        <div>
          <label htmlFor="sat-name" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">
            {t('satelliteForm.name')}
          </label>
          <input
            id="sat-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="ISS (ZARYA)"
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>

        <div className="sm:col-span-2">
          <p className="mb-1.5 text-[11px] text-lyra-mutedDim">{t('satelliteForm.tleHint')}</p>
          <div className="space-y-2 rounded border border-lyra-borderSoft bg-lyra-bgSoft/40 p-3">
            <div>
              <label htmlFor="sat-tle1" className="mb-1 block text-[10px] tracking-wider text-lyra-mutedDim">
                {t('satelliteForm.tle1')}
              </label>
              <input
                id="sat-tle1"
                value={line1}
                onChange={(e) => setLine1(e.target.value)}
                placeholder="1 25544U 98067A   24001.00000000  .00000000  00000-0  00000-0 0  9999"
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2.5 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
                required
              />
            </div>
            <div>
              <label htmlFor="sat-tle2" className="mb-1 block text-[10px] tracking-wider text-lyra-mutedDim">
                {t('satelliteForm.tle2')}
              </label>
              <input
                id="sat-tle2"
                value={line2}
                onChange={(e) => setLine2(e.target.value)}
                placeholder="2 25544  51.6400   0.0000 0000000   0.0000   0.0000 15.50000000"
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-2.5 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
                required
              />
            </div>
          </div>
        </div>
      </div>

      {error && (
        <p role="alert" className="mt-4 text-sm text-lyra-bad">
          {error}
        </p>
      )}

      <div className="mt-5 flex gap-3">
        <button
          type="submit"
          disabled={loading}
          className="rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
        >
          {loading ? t('satelliteForm.submitting') : t('satelliteForm.submit')}
        </button>
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold"
        >
          {t('common.cancel')}
        </button>
      </div>
    </form>
  )
}
