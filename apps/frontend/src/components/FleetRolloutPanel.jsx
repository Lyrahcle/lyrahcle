import { useEffect, useMemo, useState } from 'react'
import { agentRolloutAction, createAgentRollout, fetchAgentRollout, fetchAgentRollouts } from '../lib/api.js'
import ConfirmDialog from './ConfirmDialog.jsx'
import TableScroll from './TableScroll.jsx'

const STATUS_TONE = {
  applied: 'text-lyra-good border-lyra-good/30 bg-lyra-good/10',
  running: 'text-lyra-goldBright border-lyra-gold/30 bg-lyra-gold/10',
  issued: 'text-lyra-goldBright border-lyra-gold/30 bg-lyra-gold/10',
  rejected: 'text-lyra-bad border-lyra-bad/30 bg-lyra-bad/10',
  rolled_back: 'text-lyra-muted border-lyra-border bg-lyra-panel',
  paused: 'text-lyra-warn border-lyra-warn/30 bg-lyra-warn/10',
  pending: 'text-lyra-muted border-lyra-border bg-lyra-panel',
  draft: 'text-lyra-muted border-lyra-border bg-lyra-panel',
}

function StatusPill({ status }) {
  return <span className={`inline-flex rounded border px-2 py-0.5 font-mono text-[10px] tracking-wide ${STATUS_TONE[status] || STATUS_TONE.pending}`}>{String(status || 'unknown').replaceAll('_', ' ')}</span>
}

function rolloutStats(targets = []) {
  const counts = { pending: 0, issued: 0, applied: 0, rejected: 0, rolled_back: 0 }
  for (const target of targets) counts[target.status] = (counts[target.status] || 0) + 1
  const total = targets.length
  const complete = counts.applied + counts.rejected + counts.rolled_back
  const progress = total ? Math.round((complete / total) * 100) : 0
  return { ...counts, total, complete, progress }
}

function currentBatch(targets = [], batchSize = 1) {
  const active = targets.filter((target) => target.status === 'issued')
  if (!active.length) return null
  const firstOrdinal = Math.min(...active.map((target) => Number(target.ordinal || 0)))
  return Math.floor(firstOrdinal / Math.max(1, Number(batchSize || 1))) + 1
}

function latestIssuedAt(targets = []) {
  const timestamps = targets.map((target) => target.issued_at ? new Date(target.issued_at).getTime() : 0).filter(Boolean)
  return timestamps.length ? Math.max(...timestamps) : null
}


function fleetAgentEligibility(agent, frequencyHz) {
  if (agent.status !== 'active') return { eligible: false, reason: 'Agent is not active' }
  const device = agent.device
  if (!device) return { eligible: false, reason: 'Hardware capability report required' }
  if (device.compatibility !== 'compatible') return { eligible: false, reason: device.compatibility_reason || `Hardware is ${device.compatibility || 'not compatible'}` }
  const caps = device.capabilities || {}
  if (caps.integrity_status === 'changed') return { eligible: false, reason: 'Edge integrity manifest changed from its baseline' }
  const min = Number(caps.frequency_min_hz || 0)
  const max = Number(caps.frequency_max_hz || 0)
  const frequency = Number(frequencyHz)
  if (!Number.isFinite(frequency) || frequency < min || frequency > max) return { eligible: false, reason: `Frequency outside reported range ${min}-${max} Hz` }
  return { eligible: true, reason: 'Eligible for staged rollout' }
}

function oldestActiveIssuedAt(targets = []) {
  const timestamps = targets.filter((target) => target.status === 'issued' && target.issued_at).map((target) => new Date(target.issued_at).getTime()).filter(Boolean)
  return timestamps.length ? Math.min(...timestamps) : null
}

function AckDeadlineStatus({ rollout, now }) {
  const issuedAt = oldestActiveIssuedAt(rollout.targets)
  const timeoutSeconds = Number(rollout.ack_timeout_seconds || 900)
  if (!issuedAt || !timeoutSeconds || rollout.status !== 'running') return null
  const remaining = Math.max(0, Math.ceil((issuedAt + timeoutSeconds * 1000 - now) / 1000))
  const mins = Math.floor(remaining / 60)
  const secs = remaining % 60
  const expired = remaining === 0
  return <div className={`rounded border px-3 py-2 ${expired ? 'border-lyra-bad/40 bg-lyra-bad/10' : 'border-lyra-borderSoft bg-lyra-bg/30'}`} aria-label="ACK deadline">
    <p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">ACK deadline</p>
    <p className={`mt-1 font-mono text-sm ${expired ? 'text-lyra-bad' : 'text-lyra-text'}`}>{expired ? 'Expired · fail closed' : `${mins}:${String(secs).padStart(2, '0')} remaining`}</p>
  </div>
}

function SoakStatus({ rollout, now }) {
  const issuedAt = latestIssuedAt(rollout.targets)
  if (!issuedAt || !rollout.soak_seconds || rollout.status !== 'running') return null
  const elapsed = Math.max(0, Math.floor((now - issuedAt) / 1000))
  const remaining = Math.max(0, rollout.soak_seconds - elapsed)
  const mins = Math.floor(remaining / 60)
  const secs = remaining % 60
  return <div className="rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2" aria-label="Soak timer">
    <p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">Soak timer</p>
    <p className="mt-1 font-mono text-sm text-lyra-text">{remaining === 0 ? 'Ready to promote' : `${mins}:${String(secs).padStart(2, '0')} remaining`}</p>
  </div>
}

export default function FleetRolloutPanel({ agents, satellites = [], canManage = true }) {
  const activeAgents = useMemo(() => agents.filter((a) => a.status === 'active'), [agents])
  const [items, setItems] = useState([])
  const [selected, setSelected] = useState(null)
  const [name, setName] = useState('Canary rollout')
  const [frequencyHz, setFrequencyHz] = useState('437500000')
  const [intervalSeconds, setIntervalSeconds] = useState('10')
  const [satelliteId, setSatelliteId] = useState('')
  const [batchSize, setBatchSize] = useState('1')
  const [failureThreshold, setFailureThreshold] = useState('20')
  const [soakSeconds, setSoakSeconds] = useState('300')
  const [ackTimeoutSeconds, setAckTimeoutSeconds] = useState('900')
  const [agentIds, setAgentIds] = useState([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const [now, setNow] = useState(Date.now())
  const [pendingRollback, setPendingRollback] = useState(false)

  const policyByAgent = useMemo(() => new Map(activeAgents.map((agent) => [agent.agent_id, fleetAgentEligibility(agent, frequencyHz)])), [activeAgents, frequencyHz])
  const eligibleAgents = useMemo(() => activeAgents.filter((agent) => policyByAgent.get(agent.agent_id)?.eligible), [activeAgents, policyByAgent])

  async function reload(selectId = selected?.id) {
    const list = await fetchAgentRollouts()
    setItems(list)
    if (selectId) setSelected(await fetchAgentRollout(selectId))
  }
  useEffect(() => { reload().catch((e) => setError(e.message)) }, [])
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [])
  useEffect(() => {
    setAgentIds((prev) => prev.filter((id) => policyByAgent.get(id)?.eligible))
  }, [policyByAgent])

  function toggleAgent(id) { setAgentIds((prev) => prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]) }
  async function create(e) {
    e.preventDefault(); setBusy(true); setError(null)
    try {
      const created = await createAgentRollout({ name, agent_ids: agentIds, config: { frequency_hz: Number(frequencyHz), capture_interval_seconds: Number(intervalSeconds), satellite_id: satelliteId || null }, batch_size: Number(batchSize), failure_threshold_percent: Number(failureThreshold), soak_seconds: Number(soakSeconds), ack_timeout_seconds: Number(ackTimeoutSeconds) })
      await reload(created.id)
    } catch (e) { setError(e.message) } finally { setBusy(false) }
  }
  async function act(action) {
    if (!selected) return
    if (action === 'rollback') { setPendingRollback(true); return }
    setBusy(true); setError(null)
    try { const updated = await agentRolloutAction(selected.id, action); setSelected(updated); await reload(updated.id) } catch (e) { setError(e.message); await reload(selected.id).catch(() => {}) } finally { setBusy(false) }
  }

  async function confirmRollback() {
    if (!selected) return
    setBusy(true); setError(null)
    try { const updated = await agentRolloutAction(selected.id, 'rollback'); setSelected(updated); setPendingRollback(false); await reload(updated.id) } catch (e) { setError(e.message) } finally { setBusy(false) }
  }

  const stats = rolloutStats(selected?.targets)
  const batch = selected ? currentBatch(selected.targets, selected.batch_size) : null
  const decidedTargets = stats.applied + stats.rejected
  const rejectionRate = decidedTargets ? Math.floor((stats.rejected / decidedTargets) * 100) : 0
  const thresholdExceeded = selected && stats.rejected > 0 && rejectionRate > Number(selected.failure_threshold_percent || 0)
  const activeIssuedAt = selected ? oldestActiveIssuedAt(selected.targets) : null
  const ackExpired = Boolean(selected && activeIssuedAt && now >= activeIssuedAt + Number(selected.ack_timeout_seconds || 900) * 1000)

  return <section className="mt-8 border-t border-lyra-borderSoft pt-5" aria-labelledby="fleet-rollout-title">
    <div className="flex flex-wrap items-center justify-between gap-2"><div><h3 id="fleet-rollout-title" className="font-mono text-sm tracking-wider text-lyra-gold">Fleet staged rollouts</h3><p className="mt-1 text-sm text-lyra-muted">Issue signed configuration as operator-promoted canary batches. Promotion revalidates SDR hardware/integrity policy, blocks while a batch is unacknowledged, and fails closed on rejection or ACK timeout.</p></div></div>
    {canManage && <form onSubmit={create} className="mt-4 grid gap-3 rounded border border-lyra-borderSoft p-3 md:grid-cols-3">
      <label className="grid gap-1 text-xs text-lyra-muted">Name<input value={name} onChange={(e)=>setName(e.target.value)} required maxLength={120} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Frequency Hz<input type="number" min="100000" max="6000000000" value={frequencyHz} onChange={(e)=>setFrequencyHz(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Capture interval (s)<input type="number" min="1" max="3600" value={intervalSeconds} onChange={(e)=>setIntervalSeconds(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Satellite<select value={satelliteId} onChange={(e)=>setSatelliteId(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text"><option value="">None</option>{satellites.map((s)=><option key={s.id} value={s.id}>{s.name}</option>)}</select></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Batch size<input type="number" min="1" max={Math.max(1, eligibleAgents.length)} value={batchSize} onChange={(e)=>setBatchSize(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Failure threshold %<input type="number" min="0" max="100" value={failureThreshold} onChange={(e)=>setFailureThreshold(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Soak target (s)<input type="number" min="0" max="86400" value={soakSeconds} onChange={(e)=>setSoakSeconds(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">ACK timeout (s)<input type="number" min="30" max="86400" value={ackTimeoutSeconds} onChange={(e)=>setAckTimeoutSeconds(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <fieldset className="md:col-span-2"><legend className="text-xs text-lyra-muted">Fleet policy · {eligibleAgents.length}/{activeAgents.length} eligible</legend><p className="mt-1 text-[11px] text-lyra-mutedDim">Requires a fresh compatible RX report, unchanged integrity baseline, and frequency coverage.</p><div className="mt-2 flex max-h-36 flex-wrap gap-2 overflow-auto">{activeAgents.map((a)=>{ const policy = policyByAgent.get(a.agent_id); return <label key={a.agent_id} title={policy?.reason} className={`rounded border px-2 py-1 font-mono text-xs ${policy?.eligible ? 'border-lyra-borderSoft text-lyra-muted' : 'border-lyra-bad/30 text-lyra-mutedDim'}`}><input type="checkbox" className="mr-1" disabled={!policy?.eligible} checked={agentIds.includes(a.agent_id)} onChange={()=>toggleAgent(a.agent_id)} /><span>{a.agent_id}</span><span className="ml-1 text-[10px]">{policy?.eligible ? 'eligible' : 'blocked'}</span></label>})}</div></fieldset>
      <button disabled={busy || agentIds.length===0} className="rounded bg-lyra-gold px-3 py-2 text-sm text-lyra-bg disabled:opacity-50">Create staged rollout</button>
    </form>}
    {error && <p role="alert" className="mt-3 text-sm text-lyra-bad">{error}</p>}
    <div className="mt-4 grid gap-4 lg:grid-cols-[1fr_1.4fr]">
      <TableScroll><table className="w-full text-sm"><thead><tr><th className="pb-2 text-left text-xs text-lyra-muted">Rollout</th><th className="pb-2 text-left text-xs text-lyra-muted">Status</th></tr></thead><tbody>{items.map((r)=><tr key={r.id} className="border-t border-lyra-borderSoft"><td className="py-2"><button onClick={()=>fetchAgentRollout(r.id).then(setSelected).catch((e)=>setError(e.message))} className="text-left text-lyra-goldBright hover:underline">{r.name}</button></td><td><StatusPill status={r.status} /></td></tr>)}</tbody></table></TableScroll>
      <div className="rounded border border-lyra-borderSoft p-3">{selected ? <><div className="flex flex-wrap items-start justify-between gap-3"><div><div className="flex flex-wrap items-center gap-2"><p className="font-medium text-lyra-text">{selected.name}</p><StatusPill status={selected.status} /></div><p className="mt-1 font-mono text-xs text-lyra-muted">batch {selected.batch_size} · reject &gt; {selected.failure_threshold_percent}% · soak {selected.soak_seconds}s · ACK {selected.ack_timeout_seconds || 900}s</p></div>{canManage && <div className="flex flex-wrap gap-2">{selected.status==='draft'&&<button disabled={busy} onClick={()=>act('start')} className="rounded border border-lyra-gold px-2 py-1 text-xs text-lyra-gold disabled:opacity-50">Start canary</button>}{selected.status==='running'&&<><button disabled={busy || thresholdExceeded || (stats.issued > 0 && !ackExpired)} title={ackExpired ? 'Acknowledgement deadline exceeded; this action will fail the rollout closed' : stats.issued > 0 ? 'Wait for the active batch to acknowledge' : thresholdExceeded ? 'Rejection threshold exceeded' : ''} onClick={()=>act('advance')} className="rounded border border-lyra-gold px-2 py-1 text-xs text-lyra-gold disabled:opacity-40">{ackExpired ? 'Fail overdue batch' : 'Advance batch'}</button><button disabled={busy} onClick={()=>act('pause')} className="rounded border border-lyra-border px-2 py-1 text-xs text-lyra-muted disabled:opacity-50">Pause</button></>}{selected.status==='paused'&&<button disabled={busy} onClick={()=>act('resume')} className="rounded border border-lyra-gold px-2 py-1 text-xs text-lyra-gold disabled:opacity-50">Resume</button>}{!['rolled_back','draft'].includes(selected.status)&&<button disabled={busy} onClick={()=>act('rollback')} className="rounded border border-lyra-bad px-2 py-1 text-xs text-lyra-bad disabled:opacity-50">Rollback</button>}</div>}</div>
        <div className="mt-4 grid gap-2 sm:grid-cols-2 xl:grid-cols-5">
          <div className="rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2"><p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">Progress</p><p className="mt-1 font-mono text-sm text-lyra-text">{stats.complete}/{stats.total} · {stats.progress}%</p></div>
          <div className="rounded border border-lyra-borderSoft bg-lyra-bg/30 px-3 py-2"><p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">Current batch</p><p className="mt-1 font-mono text-sm text-lyra-text">{batch ? `#${batch}` : '—'}</p></div>
          <div className={`rounded border px-3 py-2 ${thresholdExceeded ? 'border-lyra-bad/40 bg-lyra-bad/10' : 'border-lyra-borderSoft bg-lyra-bg/30'}`}><p className="text-[10px] tracking-[0.15em] text-lyra-mutedDim">Rejections</p><p className={`mt-1 font-mono text-sm ${thresholdExceeded ? 'text-lyra-bad' : 'text-lyra-text'}`}>{stats.rejected}/{decidedTargets || 0} · {rejectionRate}%</p></div>
          <SoakStatus rollout={selected} now={now} />
          <AckDeadlineStatus rollout={selected} now={now} />
        </div>
        <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-lyra-borderSoft" aria-label={`Rollout progress ${stats.progress}%`}><div className="h-full bg-lyra-gold transition-[width]" style={{ width: `${stats.progress}%` }} /></div>
        {thresholdExceeded && <p role="alert" className="mt-3 rounded border border-lyra-bad/30 bg-lyra-bad/10 px-3 py-2 text-xs text-lyra-bad">Promotion is blocked because the rejection rate exceeds this rollout's failure threshold.</p>}
        {ackExpired && <p role="alert" className="mt-3 rounded border border-lyra-bad/30 bg-lyra-bad/10 px-3 py-2 text-xs text-lyra-bad">The active batch exceeded its acknowledgement deadline. Use Fail overdue batch to reconcile the server state to failed.</p>}
        <div className="mt-3 space-y-1">{selected.targets?.map((t)=><div key={t.agent_id} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-t border-lyra-borderSoft py-2 font-mono text-xs"><div className="min-w-0"><span className="block truncate text-lyra-text">{t.agent_id}</span><span className="text-[10px] text-lyra-mutedDim">cohort {Math.floor(Number(t.ordinal || 0) / Math.max(1, Number(selected.batch_size || 1))) + 1}{t.config_version ? ` · config v${t.config_version}` : ''}</span></div><StatusPill status={t.status} /></div>)}</div></> : <p className="text-sm text-lyra-muted">Select a rollout to inspect targets and promote it.</p>}</div>
    </div>
    {pendingRollback && (
      <ConfirmDialog
        title="Rollback rollout"
        message={`Rollback ${selected?.name}? A compensating config will be issued to eligible targets.`}
        confirmLabel="Rollback"
        busy={busy}
        onConfirm={confirmRollback}
        onCancel={() => setPendingRollback(false)}
      />
    )}
  </section>
}
