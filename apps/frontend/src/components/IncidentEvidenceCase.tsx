import { useMemo, useState } from 'react'
import { FilePlus2 } from 'lucide-react'
import { addIncidentEvidence, addIncidentNote, createIncident } from '../lib/api'
import { buildCrossDomainAttackChain } from './CrossDomainAttackChain'

type Entity = Record<string, any>

export function buildIncidentEvidencePackage({
  satellite,
  anomalies = [],
  incidents = [],
  orbitalEvents = [],
  tmtcAnalysis = null,
}: {
  satellite: Entity
  anomalies?: Entity[]
  incidents?: Entity[]
  orbitalEvents?: Entity[]
  tmtcAnalysis?: any
}) {
  const stages = buildCrossDomainAttackChain({ anomalies, incidents, orbitalEvents, tmtcAnalysis })
  const observed = stages.filter((stage) => stage.state !== 'unavailable')
  const evidence: { type: string; reference: string; label: string }[] = []

  anomalies.slice(0, 8).forEach((item) => {
    if (!item.id) return
    evidence.push({
      type: 'rf_anomaly',
      reference: String(item.id),
      label: `${item.metric || 'RF anomaly'}${item.z_score != null ? ` · ${Number(item.z_score).toFixed(1)}σ` : ''}`,
    })
  })

  orbitalEvents.slice(0, 8).forEach((item) => {
    if (!item.id) return
    evidence.push({
      type: 'orbital_event',
      reference: String(item.id),
      label: `${item.event_type || 'orbital'} candidate`,
    })
  })

  const replayCount = Number(tmtcAnalysis?.replay_candidates || 0)
  if (replayCount > 0) {
    evidence.push({
      type: 'tmtc_session',
      reference: `session-replay-candidates:${replayCount}`,
      label: `${replayCount} CCSDS replay candidate${replayCount === 1 ? '' : 's'} from current operator session`,
    })
  }

  return {
    title: `${satellite.name || satellite.id} cross-domain security case`,
    description: `Operator-created case for ${satellite.name || satellite.id}. ${observed.length} evidence domain${observed.length === 1 ? '' : 's'} currently contain candidate or observed evidence. This record does not assert causality, attribution, malicious intent, or a confirmed attack.`,
    severity: observed.some((stage) => stage.state === 'observed') ? 'high' : observed.length >= 2 ? 'medium' : 'low',
    evidence,
    note: `Cross-domain snapshot: ${stages.map((stage) => `${stage.title}=${stage.state}`).join('; ')}.`,
  }
}

export default function IncidentEvidenceCase({
  satellite,
  anomalies = [],
  incidents = [],
  orbitalEvents = [],
  tmtcAnalysis = null,
  canManage = false,
  agentId = '',
  onCreated = () => {},
}: {
  satellite: Entity
  anomalies?: Entity[]
  incidents?: Entity[]
  orbitalEvents?: Entity[]
  tmtcAnalysis?: any
  canManage?: boolean
  agentId?: string
  onCreated?: (incident: any) => void
}) {
  const [status, setStatus] = useState<'idle' | 'creating' | 'done' | 'error'>('idle')
  const pkg = useMemo(() => buildIncidentEvidencePackage({ satellite, anomalies, incidents, orbitalEvents, tmtcAnalysis }), [satellite, anomalies, incidents, orbitalEvents, tmtcAnalysis])

  if (!canManage) return null

  async function createCase() {
    setStatus('creating')
    try {
      const incident = await createIncident({
        title: pkg.title,
        description: pkg.description,
        severity: pkg.severity,
        satelliteId: satellite.id,
        agentId: agentId || undefined,
      })
      const incidentId = incident?.id
      if (!incidentId) throw new Error('incident id missing')
      for (const item of pkg.evidence) await addIncidentEvidence(incidentId, item)
      await addIncidentNote(incidentId, pkg.note)
      setStatus('done')
      onCreated(incident)
    } catch {
      setStatus('error')
    }
  }

  return <div className="rounded-xl border border-lyra-border bg-lyra-panel/40 p-5">
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">CASE WORKFLOW</p>
        <h2 className="mt-1 font-display text-xl text-lyra-text">Create reproducible incident case</h2>
        <p className="mt-2 max-w-3xl text-sm text-lyra-muted">Creates an operator-owned incident and attaches the currently available RF, orbital and TM/TC evidence references. Candidate evidence remains explicitly non-confirmatory.</p>
      </div>
      <button type="button" onClick={createCase} disabled={status === 'creating'} className="inline-flex items-center justify-center gap-2 rounded border border-lyra-gold/40 bg-lyra-gold/10 px-4 py-2 text-sm text-lyra-goldBright disabled:opacity-50">
        <FilePlus2 className="h-4 w-4" />
        {status === 'creating' ? 'Creating…' : 'Create incident case'}
      </button>
    </div>
    <div className="mt-4 flex flex-wrap gap-2 text-xs text-lyra-muted">
      <span className="rounded border border-lyra-borderSoft px-2 py-1">{pkg.evidence.length} evidence references</span>
      <span className="rounded border border-lyra-borderSoft px-2 py-1">severity {pkg.severity}</span>
      {status === 'done' && <span className="rounded border border-lyra-good/30 bg-lyra-good/10 px-2 py-1 text-lyra-good">Case created</span>}
      {status === 'error' && <span className="rounded border border-lyra-bad/30 bg-lyra-bad/10 px-2 py-1 text-lyra-bad">Case creation failed</span>}
    </div>
  </div>
}
