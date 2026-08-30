import { useEffect, useMemo, useState } from 'react'
import { applyAgentConfig, fetchAgentConfig, fetchAgentConfigHistory, rollbackAgentConfig } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import TableScroll from './TableScroll.jsx'

export default function AgentRemoteConfigPanel({ agents }) {
  const { t } = useI18n()
  const activeAgents = useMemo(() => agents.filter((agent) => agent.status === 'active'), [agents])
  const [agentId, setAgentId] = useState('')
  const [frequencyHz, setFrequencyHz] = useState('437500000')
  const [intervalSeconds, setIntervalSeconds] = useState('10')
  const [current, setCurrent] = useState(null)
  const [history, setHistory] = useState([])
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (!agentId && activeAgents.length) setAgentId(activeAgents[0].agent_id)
  }, [activeAgents, agentId])

  useEffect(() => {
    if (!agentId) {
      setCurrent(null)
      setHistory([])
      return
    }
    let cancelled = false
    Promise.all([fetchAgentConfig(agentId), fetchAgentConfigHistory(agentId)])
      .then(([desired, items]) => {
        if (cancelled) return
        setCurrent(desired)
        setHistory(items)
        if (desired?.config) {
          setFrequencyHz(String(desired.config.frequency_hz))
          setIntervalSeconds(String(desired.config.capture_interval_seconds))
        }
      })
      .catch((err) => {
        if (!cancelled) setError(err.message)
      })
    return () => {
      cancelled = true
    }
  }, [agentId])

  async function reload() {
    const [desired, items] = await Promise.all([fetchAgentConfig(agentId), fetchAgentConfigHistory(agentId)])
    setCurrent(desired)
    setHistory(items)
  }

  async function submit(e) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await applyAgentConfig(agentId, Number(frequencyHz), Number(intervalSeconds))
      await reload()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function rollback(version) {
    setBusy(true)
    setError(null)
    try {
      await rollbackAgentConfig(agentId, version)
      await reload()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (!activeAgents.length) {
    return <p className="text-sm text-lyra-muted">{t('connectivity.remoteConfigNoAgents')}</p>
  }

  return (
    <div>
      <p className="mb-4 text-sm text-lyra-muted">{t('connectivity.remoteConfigHint')}</p>
      <form onSubmit={submit} className="grid gap-3 md:grid-cols-[1.3fr_1fr_1fr_auto] md:items-end">
        <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
          {t('connectivity.agent')}
          <select value={agentId} onChange={(e) => setAgentId(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 font-mono text-sm normal-case text-lyra-text">
            {activeAgents.map((agent) => <option key={agent.agent_id} value={agent.agent_id}>{agent.agent_id}</option>)}
          </select>
        </label>
        <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
          {t('connectivity.remoteFrequency')}
          <input type="number" min="100000" max="6000000000" required value={frequencyHz} onChange={(e) => setFrequencyHz(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 font-mono text-sm normal-case text-lyra-text" />
        </label>
        <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
          {t('connectivity.remoteInterval')}
          <input type="number" min="1" max="3600" required value={intervalSeconds} onChange={(e) => setIntervalSeconds(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-2 font-mono text-sm normal-case text-lyra-text" />
        </label>
        <button disabled={busy} className="rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">
          {busy ? t('connectivity.remoteSaving') : t('connectivity.remoteApply')}
        </button>
      </form>

      {error && <p role="alert" className="mt-3 text-sm text-lyra-bad">{error}</p>}
      {current && (
        <div className="mt-4 rounded border border-lyra-borderSoft bg-lyra-bgSoft/40 p-3 font-mono text-xs text-lyra-muted">
          {t('connectivity.remoteDesiredVersion', { version: current.version })} · {t('connectivity.remoteAck', { status: current.ack_status || 'pending' })}
          {current.acked_at && ` · ${new Date(current.acked_at).toLocaleString()}`}
        </div>
      )}

      {history.length > 0 && (
        <div className="mt-5">
          <p className="mb-2 font-mono text-[11px] uppercase tracking-wider text-lyra-gold">{t('connectivity.remoteHistory')}</p>
          <TableScroll>
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-lyra-borderSoft">
                  <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.remoteVersion')}</th>
                  <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.remoteFrequency')}</th>
                  <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.remoteInterval')}</th>
                  <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.remoteAckLabel')}</th>
                  <th className="pb-2" />
                </tr>
              </thead>
              <tbody>
                {history.map((item) => (
                  <tr key={item.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                    <td className="py-2.5 font-mono text-xs text-lyra-goldBright">v{item.version}</td>
                    <td className="py-2.5 font-mono text-xs text-lyra-text">{item.config.frequency_hz}</td>
                    <td className="py-2.5 font-mono text-xs text-lyra-text">{item.config.capture_interval_seconds}s</td>
                    <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">{item.ack_status || '—'}</td>
                    <td className="py-2.5 text-right">
                      {item.version !== current?.version && <button type="button" disabled={busy} onClick={() => rollback(item.version)} className="font-mono text-xs uppercase text-lyra-goldBright hover:underline disabled:opacity-50">{t('connectivity.remoteRollback')}</button>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
        </div>
      )}
    </div>
  )
}
