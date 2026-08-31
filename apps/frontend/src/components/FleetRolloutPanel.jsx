import { useEffect, useMemo, useState } from 'react'
import { agentRolloutAction, createAgentRollout, fetchAgentRollout, fetchAgentRollouts } from '../lib/api.js'
import TableScroll from './TableScroll.jsx'

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
  const [agentIds, setAgentIds] = useState([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  async function reload(selectId = selected?.id) {
    const list = await fetchAgentRollouts()
    setItems(list)
    if (selectId) setSelected(await fetchAgentRollout(selectId))
  }
  useEffect(() => { reload().catch((e) => setError(e.message)) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  function toggleAgent(id) { setAgentIds((prev) => prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]) }
  async function create(e) {
    e.preventDefault(); setBusy(true); setError(null)
    try {
      const created = await createAgentRollout({ name, agent_ids: agentIds, config: { frequency_hz: Number(frequencyHz), capture_interval_seconds: Number(intervalSeconds), satellite_id: satelliteId || null }, batch_size: Number(batchSize), failure_threshold_percent: Number(failureThreshold), soak_seconds: Number(soakSeconds) })
      await reload(created.id)
    } catch (e) { setError(e.message) } finally { setBusy(false) }
  }
  async function act(action) {
    if (!selected) return; setBusy(true); setError(null)
    try { const updated = await agentRolloutAction(selected.id, action); setSelected(updated); await reload(updated.id) } catch (e) { setError(e.message) } finally { setBusy(false) }
  }

  return <section className="mt-8 border-t border-lyra-borderSoft pt-5" aria-labelledby="fleet-rollout-title">
    <div className="flex flex-wrap items-center justify-between gap-2"><div><h3 id="fleet-rollout-title" className="font-mono text-sm uppercase tracking-wider text-lyra-gold">Fleet staged rollouts</h3><p className="mt-1 text-sm text-lyra-muted">Issue signed configuration as operator-promoted canary batches. Promotion blocks while a batch is unacknowledged and fails closed when the rejection threshold is exceeded.</p></div></div>
    {canManage && <form onSubmit={create} className="mt-4 grid gap-3 rounded border border-lyra-borderSoft p-3 md:grid-cols-3">
      <label className="grid gap-1 text-xs text-lyra-muted">Name<input value={name} onChange={(e)=>setName(e.target.value)} required maxLength={120} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Frequency Hz<input type="number" min="100000" max="6000000000" value={frequencyHz} onChange={(e)=>setFrequencyHz(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Capture interval (s)<input type="number" min="1" max="3600" value={intervalSeconds} onChange={(e)=>setIntervalSeconds(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Satellite<select value={satelliteId} onChange={(e)=>setSatelliteId(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text"><option value="">None</option>{satellites.map((s)=><option key={s.id} value={s.id}>{s.name}</option>)}</select></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Batch size<input type="number" min="1" value={batchSize} onChange={(e)=>setBatchSize(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Failure threshold %<input type="number" min="0" max="100" value={failureThreshold} onChange={(e)=>setFailureThreshold(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <label className="grid gap-1 text-xs text-lyra-muted">Soak target (s)<input type="number" min="0" max="86400" value={soakSeconds} onChange={(e)=>setSoakSeconds(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 text-lyra-text" /></label>
      <fieldset className="md:col-span-2"><legend className="text-xs text-lyra-muted">Active agents</legend><div className="mt-1 flex max-h-28 flex-wrap gap-2 overflow-auto">{activeAgents.map((a)=><label key={a.agent_id} className="rounded border border-lyra-borderSoft px-2 py-1 font-mono text-xs text-lyra-muted"><input type="checkbox" className="mr-1" checked={agentIds.includes(a.agent_id)} onChange={()=>toggleAgent(a.agent_id)} />{a.agent_id}</label>)}</div></fieldset>
      <button disabled={busy || agentIds.length===0} className="rounded bg-lyra-gold px-3 py-2 text-sm text-lyra-bg disabled:opacity-50">Create staged rollout</button>
    </form>}
    {error && <p role="alert" className="mt-3 text-sm text-lyra-bad">{error}</p>}
    <div className="mt-4 grid gap-4 lg:grid-cols-[1fr_1.4fr]">
      <TableScroll><table className="w-full text-sm"><thead><tr><th className="pb-2 text-left text-xs text-lyra-muted">Rollout</th><th className="pb-2 text-left text-xs text-lyra-muted">Status</th></tr></thead><tbody>{items.map((r)=><tr key={r.id} className="border-t border-lyra-borderSoft"><td className="py-2"><button onClick={()=>fetchAgentRollout(r.id).then(setSelected).catch((e)=>setError(e.message))} className="text-left text-lyra-goldBright hover:underline">{r.name}</button></td><td className="font-mono text-xs text-lyra-muted">{r.status}</td></tr>)}</tbody></table></TableScroll>
      <div className="rounded border border-lyra-borderSoft p-3">{selected ? <><div className="flex flex-wrap items-center justify-between gap-2"><div><p className="font-medium text-lyra-text">{selected.name}</p><p className="font-mono text-xs text-lyra-muted">{selected.status} · batch {selected.batch_size} · reject &gt; {selected.failure_threshold_percent}%</p></div>{canManage && <div className="flex flex-wrap gap-2">{selected.status==='draft'&&<button disabled={busy} onClick={()=>act('start')} className="rounded border border-lyra-gold px-2 py-1 text-xs text-lyra-gold">Start canary</button>}{selected.status==='running'&&<><button disabled={busy} onClick={()=>act('advance')} className="rounded border border-lyra-gold px-2 py-1 text-xs text-lyra-gold">Advance batch</button><button disabled={busy} onClick={()=>act('pause')} className="rounded border border-lyra-border px-2 py-1 text-xs text-lyra-muted">Pause</button></>}{selected.status==='paused'&&<button disabled={busy} onClick={()=>act('resume')} className="rounded border border-lyra-gold px-2 py-1 text-xs text-lyra-gold">Resume</button>}{!['rolled_back','draft'].includes(selected.status)&&<button disabled={busy} onClick={()=>act('rollback')} className="rounded border border-lyra-bad px-2 py-1 text-xs text-lyra-bad">Rollback</button>}</div>}</div><div className="mt-3 space-y-1">{selected.targets?.map((t)=><div key={t.agent_id} className="flex justify-between border-t border-lyra-borderSoft py-1 font-mono text-xs"><span>{t.agent_id}</span><span className="text-lyra-muted">{t.status}{t.config_version ? ` · v${t.config_version}` : ''}</span></div>)}</div></> : <p className="text-sm text-lyra-muted">Select a rollout to inspect targets and promote it.</p>}</div>
    </div>
  </section>
}
