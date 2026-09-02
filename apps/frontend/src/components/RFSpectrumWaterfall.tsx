import { useMemo, useState } from 'react'
import { useI18n } from '../lib/I18nContext'

type RFReading = {
  agent_id?: string
  frequency_hz?: number | string
  power_dbm?: number | string
  captured_at?: string
}

type WaterfallCell = {
  frequencyHz: number
  powerDbm: number | null
}

type WaterfallRow = {
  bucket: string
  capturedAt: string
  cells: WaterfallCell[]
}

export type WaterfallGrid = {
  frequencies: number[]
  rows: WaterfallRow[]
  minPowerDbm: number | null
  maxPowerDbm: number | null
  sampleCount: number
}

const MAX_FREQUENCIES = 32
const MAX_TIME_ROWS = 12

function finiteNumber(value: unknown): number | null {
  if (value === '' || value === null || value === undefined) return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

function minuteBucket(value: unknown): string | null {
  if (typeof value !== 'string') return null
  const parsed = Date.parse(value)
  if (!Number.isFinite(parsed)) return null
  return new Date(parsed).toISOString().slice(0, 16)
}

export function buildWaterfallGrid(readings: RFReading[] = [], agentId = 'all'): WaterfallGrid {
  const valid: Array<RFReading & { frequencyHz: number; powerDbm: number; bucket: string; capturedAt: string }> = []
  for (const reading of readings) {
    const frequencyHz = finiteNumber(reading?.frequency_hz)
    const powerDbm = finiteNumber(reading?.power_dbm)
    const bucket = minuteBucket(reading?.captured_at)
    if (frequencyHz === null || powerDbm === null || bucket === null) continue
    if (agentId !== 'all' && reading?.agent_id !== agentId) continue
    valid.push({
      ...reading,
      frequencyHz,
      powerDbm,
      bucket,
      capturedAt: new Date(String(reading.captured_at)).toISOString(),
    })
  }

  const frequencies = [...new Set(valid.map((reading) => reading.frequencyHz))]
    .sort((a, b) => a - b)
    .slice(-MAX_FREQUENCIES)
  const allowedFrequencies = new Set(frequencies)
  const buckets = [...new Set(valid.map((reading) => reading.bucket))]
    .sort((a, b) => a.localeCompare(b))
    .slice(-MAX_TIME_ROWS)
  const allowedBuckets = new Set(buckets)
  const matrix = new Map<string, { powerDbm: number; capturedAt: string }>()

  for (const reading of valid) {
    if (!allowedFrequencies.has(reading.frequencyHz) || !allowedBuckets.has(reading.bucket)) continue
    const key = `${reading.bucket}:${reading.frequencyHz}`
    const current = matrix.get(key)
    if (!current || reading.powerDbm > current.powerDbm) {
      matrix.set(key, { powerDbm: reading.powerDbm, capturedAt: reading.capturedAt })
    }
  }

  const powers = [...matrix.values()].map((cell) => cell.powerDbm)
  const rows = buckets.map((bucket) => {
    let capturedAt = `${bucket}:00.000Z`
    const cells = frequencies.map((frequencyHz) => {
      const cell = matrix.get(`${bucket}:${frequencyHz}`)
      if (cell) capturedAt = cell.capturedAt
      return { frequencyHz, powerDbm: cell?.powerDbm ?? null }
    })
    return { bucket, capturedAt, cells }
  })

  return {
    frequencies,
    rows,
    minPowerDbm: powers.length ? Math.min(...powers) : null,
    maxPowerDbm: powers.length ? Math.max(...powers) : null,
    sampleCount: valid.filter((reading) => allowedFrequencies.has(reading.frequencyHz) && allowedBuckets.has(reading.bucket)).length,
  }
}

function cellOpacity(powerDbm: number | null, minPowerDbm: number | null, maxPowerDbm: number | null): number {
  if (powerDbm === null || minPowerDbm === null || maxPowerDbm === null) return 0.04
  const span = maxPowerDbm - minPowerDbm
  if (span <= 0) return 0.75
  return 0.18 + ((powerDbm - minPowerDbm) / span) * 0.72
}

export default function RFSpectrumWaterfall({ readings = [] }: { readings?: RFReading[] }) {
  const { t } = useI18n()
  const [agentId, setAgentId] = useState('all')
  const agents = useMemo(
    () => [...new Set(readings.map((reading) => reading?.agent_id).filter((value): value is string => Boolean(value)))].sort((a, b) => a.localeCompare(b)),
    [readings],
  )
  const grid = useMemo(() => buildWaterfallGrid(readings, agentId), [readings, agentId])

  return (
    <figure>
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <figcaption className="text-xs tracking-[0.16em] text-lyra-mutedDim">{t('rfWaterfall.title')}</figcaption>
          <p className="mt-1 text-xs text-lyra-muted">{t('rfWaterfall.subtitle')}</p>
        </div>
        <label className="text-xs text-lyra-muted">
          <span className="mb-1 block">{t('rfWaterfall.agent')}</span>
          <select
            aria-label={t('rfWaterfall.agent')}
            value={agentId}
            onChange={(event) => setAgentId(event.target.value)}
            className="min-w-44 rounded border border-lyra-border bg-lyra-panel px-2 py-1.5 font-mono text-xs text-lyra-text"
          >
            <option value="all">{t('rfWaterfall.allAgents')}</option>
            {agents.map((agent) => <option key={agent} value={agent}>{agent}</option>)}
          </select>
        </label>
      </div>

      {!grid.rows.length || !grid.frequencies.length ? (
        <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center text-sm text-lyra-mutedDim">
          {t('rfWaterfall.empty')}
        </div>
      ) : (
        <>
          <div className="overflow-x-auto rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-3">
            <div
              role="img"
              aria-label={t('rfWaterfall.ariaLabel')}
              className="grid min-w-[40rem] gap-1"
              style={{ gridTemplateColumns: `5.5rem repeat(${grid.frequencies.length}, minmax(0, 1fr))` }}
            >
              <span aria-hidden="true" />
              {grid.frequencies.map((frequencyHz) => (
                <span key={frequencyHz} className="truncate text-center font-mono text-[9px] text-lyra-mutedDim">
                  {(frequencyHz / 1_000_000).toFixed(3)}
                </span>
              ))}
              {grid.rows.map((row) => (
                <div key={row.bucket} className="contents">
                  <span className="self-center font-mono text-[9px] text-lyra-mutedDim">
                    {new Date(row.capturedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                  </span>
                  {row.cells.map((cell) => (
                    <span
                      key={`${row.bucket}-${cell.frequencyHz}`}
                      className="h-5 rounded-sm bg-lyra-gold"
                      style={{ opacity: cellOpacity(cell.powerDbm, grid.minPowerDbm, grid.maxPowerDbm) }}
                      title={cell.powerDbm === null
                        ? `${(cell.frequencyHz / 1_000_000).toFixed(6)} MHz · ${t('rfWaterfall.noSample')}`
                        : `${(cell.frequencyHz / 1_000_000).toFixed(6)} MHz · ${cell.powerDbm.toFixed(1)} dBm`}
                    />
                  ))}
                </div>
              ))}
            </div>
          </div>
          <div className="mt-3 flex flex-col gap-2 text-[10px] text-lyra-mutedDim sm:flex-row sm:items-center sm:justify-between">
            <p>{t('rfWaterfall.summary', { rows: grid.rows.length, frequencies: grid.frequencies.length, samples: grid.sampleCount })}</p>
            <p>{t('rfWaterfall.method')}</p>
          </div>
        </>
      )}
    </figure>
  )
}
