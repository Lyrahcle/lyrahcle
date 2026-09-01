function extent(values, fallback = [0, 1]) {
  const finite = values.filter(Number.isFinite)
  if (!finite.length) return fallback
  const min = Math.min(...finite)
  const max = Math.max(...finite)
  if (min === max) return [min - 1, max + 1]
  return [min, max]
}

function pointsFor(values, width, height, padding = 10) {
  const [min, max] = extent(values.map((item) => item.value))
  const span = max - min || 1
  return values.map((item, index) => {
    const x = padding + (index / Math.max(1, values.length - 1)) * (width - padding * 2)
    const y = padding + ((max - item.value) / span) * (height - padding * 2)
    return { ...item, x, y }
  })
}

function EmptyChart({ children }) {
  return <div className="flex h-44 items-center justify-center text-sm text-lyra-mutedDim">{children}</div>
}

function SparkLine({ values, label, suffix = '' }) {
  const width = 520
  const height = 170
  if (!values.length) return <EmptyChart>No telemetry yet.</EmptyChart>
  const points = pointsFor(values, width, height, 14)
  const polyline = points.map(({ x, y }) => `${x},${y}`).join(' ')
  const latest = values.at(-1)?.value

  return (
    <figure>
      <div className="mb-3 flex items-end justify-between gap-4">
        <figcaption className="text-xs tracking-[0.16em] text-lyra-mutedDim">{label}</figcaption>
        <span className="font-mono text-sm text-lyra-goldBright">{Number.isFinite(latest) ? `${latest.toFixed(1)}${suffix}` : '—'}</span>
      </div>
      <svg viewBox={`0 0 ${width} ${height}`} className="h-44 w-full overflow-visible" role="img" aria-label={label}>
        {[0.25, 0.5, 0.75].map((ratio) => (
          <line key={ratio} x1="14" x2={width - 14} y1={height * ratio} y2={height * ratio} stroke="rgb(var(--lyra-border-soft))" strokeWidth="1" />
        ))}
        <polyline points={polyline} fill="none" stroke="rgb(var(--lyra-gold-bright))" strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" />
        {points.map((point, index) => (
          <circle key={`${point.x}-${index}`} cx={point.x} cy={point.y} r={index === points.length - 1 ? 4 : 2} fill="rgb(var(--lyra-gold-bright))">
            <title>{`${point.label}: ${point.value.toFixed(1)}${suffix}`}</title>
          </circle>
        ))}
      </svg>
    </figure>
  )
}

export function RFPowerChart({ readings }) {
  const ordered = [...(readings ?? [])]
    .filter((reading) => Number.isFinite(Number(reading.power_dbm)))
    .sort((a, b) => new Date(a.captured_at) - new Date(b.captured_at))
    .slice(-40)
    .map((reading) => ({
      value: Number(reading.power_dbm),
      label: `${reading.agent_id} · ${new Date(reading.captured_at).toLocaleTimeString()}`,
    }))
  return <SparkLine values={ordered} label="RF power trend" suffix=" dBm" />
}

export function IncidentSeverityChart({ incidents }) {
  const counts = { critical: 0, high: 0, medium: 0, low: 0 }
  for (const incident of incidents ?? []) {
    const key = String(incident.severity ?? '').toLowerCase()
    if (key in counts) counts[key] += 1
  }
  const max = Math.max(1, ...Object.values(counts))
  return (
    <figure>
      <figcaption className="mb-4 text-xs tracking-[0.16em] text-lyra-mutedDim">Current incident severity mix</figcaption>
      <div className="grid h-44 grid-cols-4 items-end gap-4" role="img" aria-label="Incidents grouped by severity">
        {Object.entries(counts).map(([severity, count]) => (
          <div key={severity} className="flex h-full flex-col justify-end gap-2">
            <div className="flex min-h-0 flex-1 items-end rounded-sm bg-lyra-bgSoft px-2">
              <div
                className="w-full rounded-t-sm bg-lyra-gold/70 transition-[height]"
                style={{ height: `${Math.max(count ? 8 : 2, (count / max) * 100)}%` }}
                title={`${severity}: ${count}`}
              />
            </div>
            <div className="text-center">
              <div className="font-mono text-sm text-lyra-text">{count}</div>
              <div className="text-[10px] tracking-wider text-lyra-mutedDim">{severity}</div>
            </div>
          </div>
        ))}
      </div>
    </figure>
  )
}

export function CVESeverityChart({ cves }) {
  const counts = { critical: 0, high: 0, medium: 0, low: 0, unknown: 0 }
  for (const cve of cves ?? []) {
    const key = String(cve.severity ?? 'unknown').toLowerCase()
    counts[key in counts ? key : 'unknown'] += 1
  }
  const total = Object.values(counts).reduce((sum, value) => sum + value, 0)
  return (
    <figure>
      <figcaption className="mb-4 text-xs tracking-[0.16em] text-lyra-mutedDim">Visible CVEs by severity</figcaption>
      {!total ? <EmptyChart>No CVEs in the current result set.</EmptyChart> : (
        <div className="space-y-3">
          {Object.entries(counts).map(([severity, count]) => (
            <div key={severity} className="grid grid-cols-[5rem_1fr_2rem] items-center gap-3">
              <span className="text-[10px] tracking-wider text-lyra-mutedDim">{severity}</span>
              <div className="h-3 overflow-hidden rounded-full bg-lyra-bgSoft">
                <div className="h-full rounded-full bg-lyra-gold/70" style={{ width: `${(count / total) * 100}%` }} />
              </div>
              <span className="text-right font-mono text-xs text-lyra-text">{count}</span>
            </div>
          ))}
        </div>
      )}
    </figure>
  )
}

export function RFSpectrumChart({ readings }) {
  const buckets = new Map()
  for (const reading of readings ?? []) {
    const hz = Number(reading.frequency_hz)
    const power = Number(reading.power_dbm)
    if (!Number.isFinite(hz) || !Number.isFinite(power)) continue
    const mhz = hz / 1e6
    const key = mhz.toFixed(3)
    const current = buckets.get(key)
    if (!current || power > current.power) buckets.set(key, { frequency: mhz, power })
  }
  const rows = [...buckets.values()].sort((a, b) => a.frequency - b.frequency).slice(-48)
  if (!rows.length) return <EmptyChart>No RF spectrum data yet.</EmptyChart>
  const minPower = Math.min(...rows.map((row) => row.power))
  const maxPower = Math.max(...rows.map((row) => row.power))
  const span = maxPower - minPower || 1

  return (
    <figure>
      <div className="mb-4 flex items-end justify-between gap-4">
        <figcaption className="text-xs tracking-[0.16em] text-lyra-mutedDim">RF spectrum snapshot</figcaption>
        <span className="font-mono text-xs text-lyra-mutedDim">{rows[0].frequency.toFixed(3)}–{rows.at(-1).frequency.toFixed(3)} MHz</span>
      </div>
      <div className="flex h-44 items-end gap-1" role="img" aria-label="RF spectrum power by frequency">
        {rows.map((row) => (
          <div key={row.frequency} className="group flex min-w-0 flex-1 items-end self-stretch">
            <div
              className="w-full rounded-t-sm bg-lyra-gold/70"
              style={{ height: `${Math.max(3, ((row.power - minPower) / span) * 100)}%` }}
              title={`${row.frequency.toFixed(3)} MHz · ${row.power.toFixed(1)} dBm`}
            />
          </div>
        ))}
      </div>
      <div className="mt-2 flex justify-between font-mono text-[10px] text-lyra-mutedDim">
        <span>{rows[0].frequency.toFixed(3)} MHz</span>
        <span>{rows.at(-1).frequency.toFixed(3)} MHz</span>
      </div>
    </figure>
  )
}

export function AnomalyTimelineChart({ anomalies }) {
  const ordered = [...(anomalies ?? [])]
    .filter((item) => item.detected_at)
    .sort((a, b) => new Date(a.detected_at) - new Date(b.detected_at))
    .slice(-36)
    .map((item) => ({
      value: Math.abs(Number(item.z_score) || 0),
      label: `${item.agent_id} · ${new Date(item.detected_at).toLocaleTimeString()}`,
    }))
  return <SparkLine values={ordered} label="Anomaly intensity trend" suffix="σ" />
}

function dayKey(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function shortDayLabel(key) {
  const [year, month, day] = key.split('-').map(Number)
  return new Date(year, month - 1, day).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

function TimeBucketBars({ items, timestampKey, label, emptyText }) {
  const counts = new Map()
  for (const item of items ?? []) {
    const key = dayKey(item?.[timestampKey])
    if (!key) continue
    counts.set(key, (counts.get(key) ?? 0) + 1)
  }
  const rows = [...counts.entries()].sort(([a], [b]) => a.localeCompare(b)).slice(-14)
  if (!rows.length) return <EmptyChart>{emptyText}</EmptyChart>
  const max = Math.max(1, ...rows.map(([, count]) => count))
  return (
    <figure>
      <figcaption className="mb-4 text-xs tracking-[0.16em] text-lyra-mutedDim">{label}</figcaption>
      <div className="flex h-44 items-end gap-2" role="img" aria-label={label}>
        {rows.map(([key, count]) => (
          <div key={key} className="flex min-w-0 flex-1 flex-col justify-end gap-2 self-stretch">
            <div className="flex min-h-0 flex-1 items-end">
              <div className="w-full rounded-t-sm bg-lyra-gold/70" style={{ height: `${Math.max(4, (count / max) * 100)}%` }} title={`${shortDayLabel(key)} · ${count}`} />
            </div>
            <span className="truncate text-center font-mono text-[9px] text-lyra-mutedDim">{shortDayLabel(key)}</span>
          </div>
        ))}
      </div>
    </figure>
  )
}

export function IncidentTimelineChart({ incidents }) {
  return <TimeBucketBars items={incidents} timestampKey="created_at" label="Incidents created over time" emptyText="No incident timeline data yet." />
}

export function CVETimelineChart({ cves }) {
  return <TimeBucketBars items={cves} timestampKey="published" label="CVE publication timeline" emptyText="No CVE timeline data in the current result set." />
}

export function OrbitalAltitudeChart({ satellites }) {
  const rows = (satellites ?? [])
    .filter((satellite) => Number.isFinite(Number(satellite.position?.altitude_km)))
    .map((satellite) => ({
      name: satellite.name,
      altitude: Number(satellite.position.altitude_km),
    }))
    .sort((a, b) => b.altitude - a.altitude)
    .slice(0, 16)

  if (!rows.length) return <EmptyChart>No live orbital altitude data yet.</EmptyChart>
  const max = Math.max(1, ...rows.map((row) => row.altitude))

  return (
    <figure>
      <div className="mb-4 flex items-end justify-between gap-4">
        <figcaption className="text-xs tracking-[0.16em] text-lyra-mutedDim">Orbital altitude profile</figcaption>
        <span className="font-mono text-xs text-lyra-mutedDim">Top {rows.length} tracked objects</span>
      </div>
      <div className="space-y-2" role="img" aria-label="Tracked satellites by current altitude">
        {rows.map((row) => (
          <div key={row.name} className="grid grid-cols-[8rem_1fr_4.5rem] items-center gap-3">
            <span className="truncate font-mono text-[10px] text-lyra-muted">{row.name}</span>
            <div className="h-2 overflow-hidden rounded-full bg-lyra-bgSoft">
              <div className="h-full rounded-full bg-lyra-gold/70" style={{ width: `${Math.max(2, (row.altitude / max) * 100)}%` }} />
            </div>
            <span className="text-right font-mono text-[10px] text-lyra-goldBright">{row.altitude.toFixed(0)} km</span>
          </div>
        ))}
      </div>
    </figure>
  )
}
