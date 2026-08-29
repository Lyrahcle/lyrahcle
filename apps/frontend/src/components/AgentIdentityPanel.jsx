import { useState } from 'react'
import { createAgentEnrollmentToken, updateAgentIdentityStatus } from '../lib/api.js'
import TableScroll from './TableScroll.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

export default function AgentIdentityPanel({ agents, onChanged }) {
  const { t } = useI18n()
  const [agentId, setAgentId] = useState('')
  const [issued, setIssued] = useState(null)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  async function issue(e) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      const result = await createAgentEnrollmentToken(agentId.trim())
      setIssued(result)
      setAgentId('')
      await onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function changeStatus(id, status) {
    setError(null)
    try {
      await updateAgentIdentityStatus(id, status)
      await onChanged()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      <div className="mb-4 rounded border border-lyra-borderSoft bg-lyra-bgSoft/40 p-3 text-xs text-lyra-muted">
        {t('connectivity.mtlsHint')}
      </div>

      <form onSubmit={issue} className="mb-5 flex flex-wrap items-end gap-3">
        <div>
          <label htmlFor="mtls-agent-id" className="mb-1 block text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            Agent ID
          </label>
          <input
            id="mtls-agent-id"
            value={agentId}
            onChange={(e) => setAgentId(e.target.value)}
            placeholder="sdr-agent-madrid"
            className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1.5 font-mono text-sm text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>
        <button
          type="submit"
          disabled={busy}
          className="rounded bg-lyra-gold px-4 py-1.5 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {busy ? t('connectivity.issuingEnrollmentToken') : t('connectivity.issueEnrollmentToken')}
        </button>
      </form>

      {error && <p role="alert" className="mb-4 text-sm text-lyra-bad">{error}</p>}

      {issued && (
        <div role="status" className="mb-5 rounded border border-lyra-good/30 bg-lyra-good/10 p-3">
          <p className="text-xs text-lyra-good">{t('connectivity.enrollmentTokenShownOnce', { agent: issued.agent_id })}</p>
          <p className="mt-1 break-all font-mono text-xs text-lyra-text">{issued.token}</p>
          <p className="mt-1 font-mono text-[10px] text-lyra-mutedDim">{t('connectivity.expires')} {new Date(issued.expires_at).toLocaleString()}</p>
        </div>
      )}

      {!agents.length ? (
        <p className="text-sm text-lyra-muted">{t('connectivity.noMtlsAgents')}</p>
      ) : (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-lyra-borderSoft">
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.agent')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.identityStatus')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.certificate')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('connectivity.lastSeen')}</th>
                <th className="pb-2" />
              </tr>
            </thead>
            <tbody>
              {agents.map((agent) => (
                <tr key={agent.agent_id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-2.5 font-mono text-xs text-lyra-goldBright">
                    {agent.agent_id}
                    {agent.agent_version && <span className="ml-2 text-lyra-mutedDim">{agent.agent_version}</span>}
                  </td>
                  <td className="py-2.5 font-mono text-xs text-lyra-text">{agent.status}</td>
                  <td className="py-2.5 font-mono text-[10px] text-lyra-mutedDim">
                    {agent.certificate ? `${agent.certificate.fingerprint_sha256.slice(0, 12)}… · ${new Date(agent.certificate.not_after).toLocaleDateString()}` : t('connectivity.pending')}
                  </td>
                  <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">{agent.last_seen_at ? new Date(agent.last_seen_at).toLocaleString() : '—'}</td>
                  <td className="py-2.5 text-right">
                    {agent.status === 'active' && <button onClick={() => changeStatus(agent.agent_id, 'quarantined')} className="mr-3 font-mono text-xs uppercase text-lyra-goldBright hover:underline">{t('connectivity.quarantine')}</button>}
                    {agent.status === 'quarantined' && <button onClick={() => changeStatus(agent.agent_id, 'active')} className="mr-3 font-mono text-xs uppercase text-lyra-good hover:underline">{t('connectivity.activate')}</button>}
                    {agent.status !== 'revoked' && <button onClick={() => changeStatus(agent.agent_id, 'revoked')} className="font-mono text-xs uppercase text-lyra-bad hover:underline">{t('connectivity.revoke')}</button>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
    </div>
  )
}
