const playbooks = [
  { title: 'Privileged account', severity: 'SEV-1 / SEV-2', action: 'Revoke sessions, disable the account, then review RBAC and audit events.', where: 'Admin / Platform' },
  { title: 'Signing key', severity: 'SEV-1', action: 'Pause rollouts, revoke the compromised KID in custody, stage and distribute a replacement.', where: 'Platform → Remote Config Signing Keys' },
  { title: 'SDR agent / edge tamper', severity: 'SEV-2', action: 'Quarantine first, preserve identity and integrity evidence, then rebuild/re-enroll if trust is lost.', where: 'Connectivity → Agent Identity / Hardware' },
  { title: 'Cross-tenant exposure', severity: 'SEV-1', action: 'Preserve request IDs and scope evidence; isolate the route or tenant. Never disable RLS as a shortcut.', where: 'Platform / Audit Log' },
  { title: 'Backend secret', severity: 'SEV-1 / SEV-2', action: 'Contain access, rotate in custody, force reauthentication when JWT trust is compromised, then verify provider health.', where: 'Platform → Secret Management' },
]

export default function IncidentResponsePanel() {
  return (
    <div className="grid gap-5">
      <div className="rounded border border-lyra-bad/30 bg-lyra-bad/5 p-4">
        <p className="font-mono text-xs uppercase tracking-wider text-lyra-bad">First 15 minutes</p>
        <p className="mt-2 text-sm text-lyra-text">Record UTC time and request IDs, preserve evidence, confirm whether the incident is active, then contain with the narrowest effective control before rotating credentials.</p>
      </div>

      <div className="grid gap-3 lg:grid-cols-2">
        {playbooks.map((item) => (
          <article key={item.title} className="rounded border border-lyra-border bg-lyra-bgSoft/50 p-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h3 className="text-sm font-medium text-lyra-text">{item.title}</h3>
              <span className="rounded border border-lyra-border px-2 py-1 font-mono text-[10px] uppercase text-lyra-goldBright">{item.severity}</span>
            </div>
            <p className="mt-2 text-xs leading-5 text-lyra-muted">{item.action}</p>
            <p className="mt-3 font-mono text-[10px] uppercase tracking-wider text-lyra-mutedDim">Act in: {item.where}</p>
          </article>
        ))}
      </div>

      <div className="rounded border border-lyra-border p-4">
        <p className="text-xs uppercase tracking-wider text-lyra-mutedDim">Recovery gates</p>
        <p className="mt-2 text-xs leading-5 text-lyra-muted">Do not close an incident until containment is verified, compromised trust is revoked/rotated, integrity is restored, regression tests exist for boundary failures, monitoring is active, and evidence/disclosure decisions are recorded.</p>
        <a className="mt-3 inline-flex text-xs font-medium text-lyra-goldBright underline underline-offset-4" href="https://github.com/mlko01/lyrahcle/blob/main/docs/incident-response-runbook.md" target="_blank" rel="noopener noreferrer">Open full incident response runbook ↗</a>
      </div>
    </div>
  )
}
