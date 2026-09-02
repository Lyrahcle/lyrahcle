type Entity = Record<string, any>

export function buildInvestigationChain(detail: any) {
  const incident = detail?.incident || {}
  const evidence = Array.isArray(detail?.evidence) ? detail.evidence : []
  const events = Array.isArray(detail?.events) ? detail.events : []

  const nodes = evidence.map((item: Entity, index: number) => ({
    id: item.id || `evidence-${index}`,
    type: item.evidence_type || 'evidence',
    reference: item.evidence_ref || '',
    label: item.label || item.evidence_type || 'Evidence',
    time: item.created_at || null,
  }))

  const history = [...events]
    .filter((item: Entity) => item.created_at || item.event_time)
    .sort((a: Entity, b: Entity) => new Date(a.created_at || a.event_time).getTime() - new Date(b.created_at || b.event_time).getTime())

  return {
    reference: incident.reference || incident.id || '—',
    nodes,
    history,
    domains: [...new Set(nodes.map((node) => node.type))],
  }
}

export default function IncidentInvestigationChain({ detail }: { detail: any }) {
  const chain = buildInvestigationChain(detail)
  return <section aria-labelledby="investigation-chain-heading" className="rounded-xl border border-lyra-border bg-lyra-panel/40 p-5">
    <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p className="text-[10px] tracking-[0.16em] text-lyra-mutedDim">INVESTIGATION</p>
        <h4 id="investigation-chain-heading" className="mt-1 font-display text-lg text-lyra-text">Evidence chain</h4>
        <p className="mt-1 text-sm text-lyra-muted">Reconstructs the evidence references and recorded incident history. Ordering is chronology, not proof of causality.</p>
      </div>
      <p className="font-mono text-xs text-lyra-goldBright">{chain.reference}</p>
    </div>

    <div className="mt-4 grid gap-3 md:grid-cols-3">
      <div className="rounded-lg border border-lyra-borderSoft p-3">
        <p className="text-[10px] tracking-wider text-lyra-mutedDim">Evidence</p>
        <p className="mt-1 font-mono text-xl text-lyra-text">{chain.nodes.length}</p>
      </div>
      <div className="rounded-lg border border-lyra-borderSoft p-3">
        <p className="text-[10px] tracking-wider text-lyra-mutedDim">Evidence types</p>
        <p className="mt-1 font-mono text-xl text-lyra-text">{chain.domains.length}</p>
      </div>
      <div className="rounded-lg border border-lyra-borderSoft p-3">
        <p className="text-[10px] tracking-wider text-lyra-mutedDim">History events</p>
        <p className="mt-1 font-mono text-xl text-lyra-text">{chain.history.length}</p>
      </div>
    </div>

    <div className="mt-5 grid gap-5 lg:grid-cols-2">
      <div>
        <p className="mb-2 text-xs tracking-wider text-lyra-muted">EVIDENCE REFERENCES</p>
        <div className="grid gap-2">
          {chain.nodes.length ? chain.nodes.map((node) => <div key={node.id} className="rounded-lg border border-lyra-borderSoft bg-lyra-bgSoft/40 p-3">
            <div className="flex items-center justify-between gap-3">
              <span className="font-mono text-[11px] text-lyra-goldBright">{node.type}</span>
              {node.time && <span className="font-mono text-[10px] text-lyra-mutedDim">{new Date(node.time).toLocaleString()}</span>}
            </div>
            <p className="mt-1 text-sm text-lyra-text">{node.label}</p>
            <p className="mt-1 break-all font-mono text-[11px] text-lyra-muted">{node.reference}</p>
          </div>) : <p className="text-sm text-lyra-muted">No evidence references recorded.</p>}
        </div>
      </div>

      <div>
        <p className="mb-2 text-xs tracking-wider text-lyra-muted">RECORDED HISTORY</p>
        <ol className="grid gap-2">
          {chain.history.length ? chain.history.map((event: Entity, index: number) => <li key={event.id || `${event.event_type || 'event'}-${index}`} className="rounded-lg border border-lyra-borderSoft p-3">
            <div className="flex items-center justify-between gap-3">
              <span className="text-sm text-lyra-text">{event.event_type || event.type || 'incident event'}</span>
              <span className="font-mono text-[10px] text-lyra-mutedDim">{new Date(event.created_at || event.event_time).toLocaleString()}</span>
            </div>
            {(event.message || event.summary || event.note) && <p className="mt-1 text-xs text-lyra-muted">{event.message || event.summary || event.note}</p>}
          </li>) : <p className="text-sm text-lyra-muted">No incident history recorded.</p>}
        </ol>
      </div>
    </div>
  </section>
}
