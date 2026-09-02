import { FormEvent, useState } from 'react'
import { analyzeCCSDSSequence } from '../lib/api'

export type Issue = {
  index: number
  apid: number
  packet_type: string
  sequence_count: number
  kind: string
  detail: string
  replay_candidate: boolean
}

export type Analysis = {
  samples: unknown[]
  issues: Issue[]
  replay_candidates: number
  sequence_gaps: number
  out_of_order: number
  collisions: number
}

function label(kind: string) {
  if (kind === 'replay_candidate') return 'Replay candidate'
  if (kind === 'sequence_gap') return 'Sequence gap'
  if (kind === 'out_of_order') return 'Out of order'
  if (kind === 'sequence_collision') return 'Sequence collision'
  return kind
}

export default function TMTCSequenceAnalyzer({ onAnalysis }: { onAnalysis?: (analysis: Analysis | null) => void }) {
  const [input, setInput] = useState('')
  const [analysis, setAnalysis] = useState<Analysis | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    const packets = input.split(/\r?\n/).map((value) => value.trim()).filter(Boolean)
    setError('')
    setAnalysis(null)
    onAnalysis?.(null)
    setLoading(true)
    try {
      const result = await analyzeCCSDSSequence(packets)
      setAnalysis(result)
      onAnalysis?.(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'CCSDS sequence analysis failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <section aria-labelledby="tmtc-sequence-analysis-title">
      <h3 id="tmtc-sequence-analysis-title" className="text-sm font-medium text-lyra-text">CCSDS sequence security analysis</h3>
      <p className="mt-1 text-xs text-lyra-muted">Paste captured CCSDS packets in chronological order, one hexadecimal packet per line. Exact repetitions are reported as replay candidates, not confirmed attacks.</p>

      <form onSubmit={submit} className="mt-4 grid gap-3">
        <label htmlFor="ccsds-sequence-input" className="text-xs text-lyra-mutedDim">Captured CCSDS packets</label>
        <textarea id="ccsds-sequence-input" rows={7} required value={input} onChange={(event) => setInput(event.target.value)} className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold" placeholder="One CCSDS packet per line" />
        <button type="submit" disabled={loading} className="w-fit rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">{loading ? 'Analyzing…' : 'Analyze sequence'}</button>
      </form>

      {error ? <p role="alert" className="mt-4 rounded border border-lyra-bad/30 bg-lyra-bad/10 px-3 py-2 text-sm text-lyra-bad">{error}</p> : null}

      {analysis ? (
        <div className="mt-5 grid gap-4">
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <Metric label="Replay candidates" value={analysis.replay_candidates} />
            <Metric label="Sequence gaps" value={analysis.sequence_gaps} />
            <Metric label="Out of order" value={analysis.out_of_order} />
            <Metric label="Collisions" value={analysis.collisions} />
          </div>
          {analysis.issues.length ? (
            <ul className="grid gap-2">
              {analysis.issues.map((issue, index) => (
                <li key={`${issue.index}-${issue.kind}-${index}`} className="rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <span className="text-xs font-medium text-lyra-text">{label(issue.kind)}</span>
                    <span className="font-mono text-[10px] text-lyra-mutedDim">APID {issue.apid} · seq {issue.sequence_count} · packet {issue.index + 1}</span>
                  </div>
                  <p className="mt-1 text-xs text-lyra-muted">{issue.detail}</p>
                </li>
              ))}
            </ul>
          ) : <p className="text-xs text-lyra-muted">No sequence anomalies detected in this capture.</p>}
          <p className="text-[10px] text-lyra-mutedDim">Sequence metadata alone cannot establish malicious intent. Correlate replay candidates with RF, station, authentication and mission evidence.</p>
        </div>
      ) : null}
    </section>
  )
}

function Metric({ label, value }: { label: string; value: number }) {
  return <div className="rounded border border-lyra-borderSoft p-3"><p className="text-[10px] text-lyra-mutedDim">{label}</p><p className="mt-1 font-mono text-lg text-lyra-text">{value}</p></div>
}
