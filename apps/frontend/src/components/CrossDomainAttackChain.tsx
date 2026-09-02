import { useMemo } from 'react'
import { classifyRFAttackCandidates } from './RFAttackTriage'

type Entity = Record<string, any>
type TMTCIssue = { kind?: string; detail?: string; replay_candidate?: boolean }
type TMTCAnalysis = { issues?: TMTCIssue[]; replay_candidates?: number; sequence_gaps?: number; out_of_order?: number; collisions?: number } | null

type Stage = {
  id: string
  title: string
  state: 'candidate' | 'observed' | 'unavailable'
  evidence: string[]
}

export function buildCrossDomainAttackChain({ anomalies = [], incidents = [], orbitalEvents = [], tmtcAnalysis = null }: {
  anomalies?: Entity[]
  incidents?: Entity[]
  orbitalEvents?: Entity[]
  tmtcAnalysis?: TMTCAnalysis
}): Stage[] {
  const rf = classifyRFAttackCandidates(anomalies)
  const jamming = rf.filter((item: any) => item.kind === 'jamming')
  const spoofing = rf.filter((item: any) => item.kind === 'spoofing')
  const tmtcIssues = tmtcAnalysis?.issues ?? []
  const replay = tmtcIssues.filter((item) => item.kind === 'replay_candidate' || item.replay_candidate)
  const maneuver = orbitalEvents.filter((item) => item.event_type === 'maneuver')
  const conjunction = orbitalEvents.filter((item) => item.event_type === 'conjunction')
  const activeIncidents = incidents.filter((item) => !['resolved', 'false_positive'].includes(String(item.status || '').toLowerCase()))

  return [
    {
      id: 'rf',
      title: 'RF signal evidence',
      state: jamming.length || spoofing.length ? 'candidate' : 'unavailable',
      evidence: [
        jamming.length ? `${jamming.length} jamming candidate${jamming.length === 1 ? '' : 's'}` : '',
        spoofing.length ? `${spoofing.length} spoofing candidate${spoofing.length === 1 ? '' : 's'}` : '',
      ].filter(Boolean),
    },
    {
      id: 'tmtc',
      title: 'TM/TC sequence evidence',
      state: tmtcAnalysis ? (tmtcIssues.length ? 'candidate' : 'observed') : 'unavailable',
      evidence: tmtcAnalysis ? [
        replay.length ? `${replay.length} replay candidate${replay.length === 1 ? '' : 's'}` : 'No replay candidate in current analyzer session',
        tmtcAnalysis.sequence_gaps ? `${tmtcAnalysis.sequence_gaps} sequence gap${tmtcAnalysis.sequence_gaps === 1 ? '' : 's'}` : '',
        tmtcAnalysis.out_of_order ? `${tmtcAnalysis.out_of_order} out-of-order packet${tmtcAnalysis.out_of_order === 1 ? '' : 's'}` : '',
        tmtcAnalysis.collisions ? `${tmtcAnalysis.collisions} sequence collision${tmtcAnalysis.collisions === 1 ? '' : 's'}` : '',
      ].filter(Boolean) : [],
    },
    {
      id: 'orbital',
      title: 'Orbital behavior evidence',
      state: maneuver.length || conjunction.length ? 'candidate' : 'unavailable',
      evidence: [
        maneuver.length ? `${maneuver.length} maneuver candidate${maneuver.length === 1 ? '' : 's'}` : '',
        conjunction.length ? `${conjunction.length} conjunction candidate${conjunction.length === 1 ? '' : 's'}` : '',
      ].filter(Boolean),
    },
    {
      id: 'incident',
      title: 'Incident response evidence',
      state: activeIncidents.length ? 'observed' : 'unavailable',
      evidence: activeIncidents.slice(0, 3).map((item) => String(item.title || item.reference || item.id || 'Open incident')),
    },
  ]
}

function badge(state: Stage['state']) {
  if (state === 'candidate') return 'border-lyra-warn/40 bg-lyra-warn/10 text-lyra-warn'
  if (state === 'observed') return 'border-lyra-good/40 bg-lyra-good/10 text-lyra-good'
  return 'border-lyra-border bg-lyra-panel text-lyra-mutedDim'
}

export default function CrossDomainAttackChain(props: {
  anomalies?: Entity[]
  incidents?: Entity[]
  orbitalEvents?: Entity[]
  tmtcAnalysis?: TMTCAnalysis
}) {
  const stages = useMemo(() => buildCrossDomainAttackChain(props), [props.anomalies, props.incidents, props.orbitalEvents, props.tmtcAnalysis])
  const active = stages.filter((stage) => stage.state !== 'unavailable').length

  return (
    <section aria-labelledby="cross-domain-chain-title" className="rounded-xl border border-lyra-border bg-lyra-panel/40 p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">CROSS-DOMAIN CORRELATION</p>
          <h2 id="cross-domain-chain-title" className="mt-1 font-display text-xl text-lyra-text">Explainable evidence chain</h2>
          <p className="mt-1 max-w-3xl text-xs text-lyra-muted">Correlates available RF, TM/TC, orbital and incident evidence without asserting causality, attribution or a confirmed attack.</p>
        </div>
        <span className="rounded border border-lyra-border px-2.5 py-1 font-mono text-[10px] text-lyra-muted">{active}/4 domains with evidence</span>
      </div>

      <ol className="mt-4 grid gap-3 lg:grid-cols-4" aria-label="Cross-domain evidence stages">
        {stages.map((stage, index) => (
          <li key={stage.id} className="relative rounded-lg border border-lyra-borderSoft bg-lyra-bg/25 p-4">
            <div className="flex items-center justify-between gap-2">
              <span className="font-mono text-[10px] text-lyra-mutedDim">0{index + 1}</span>
              <span className={`rounded border px-2 py-0.5 text-[10px] ${badge(stage.state)}`}>{stage.state}</span>
            </div>
            <h3 className="mt-3 text-sm font-medium text-lyra-text">{stage.title}</h3>
            {stage.evidence.length ? (
              <ul className="mt-2 grid gap-1 text-[11px] text-lyra-muted">
                {stage.evidence.map((item) => <li key={item}>{item}</li>)}
              </ul>
            ) : <p className="mt-2 text-[11px] text-lyra-mutedDim">No evidence available in current context.</p>}
          </li>
        ))}
      </ol>
      {props.tmtcAnalysis ? <p className="mt-3 text-[10px] text-lyra-mutedDim">TM/TC evidence comes from the current decoder session and is not automatically satellite-bound.</p> : null}
    </section>
  )
}
