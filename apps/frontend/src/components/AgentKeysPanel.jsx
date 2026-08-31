import { useState } from 'react'
import { createAgentKey, revokeAgentKey } from '../lib/api.js'
import TableScroll from './TableScroll.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

export default function AgentKeysPanel({ keys, onChanged, canManage = true }) {
  const { t } = useI18n()
  const [newAgentId, setNewAgentId] = useState('')
  const [issuedKey, setIssuedKey] = useState(null)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)

  async function handleCreate(e) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      const result = await createAgentKey(newAgentId.trim())
      setIssuedKey(result)
      setNewAgentId('')
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function handleRevoke(agentId) {
    setError(null)
    try {
      await revokeAgentKey(agentId)
      onChanged()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      {canManage && <form onSubmit={handleCreate} className="mb-5 flex flex-wrap items-end gap-3">
        <div>
          <label htmlFor="new-agent-id" className="mb-1 block text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            {t('agentKeys.newAgentId')}
          </label>
          <input
            id="new-agent-id"
            value={newAgentId}
            onChange={(e) => setNewAgentId(e.target.value)}
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
          {busy ? t('agentKeys.issuing') : t('agentKeys.issueKey')}
        </button>
      </form>}

      {error && (
        <p role="alert" className="mb-4 text-sm text-lyra-bad">
          {error}
        </p>
      )}

      {issuedKey && (
        <div role="status" className="mb-5 rounded border border-lyra-good/30 bg-lyra-good/10 p-3">
          <p className="text-xs text-lyra-good">
            {t('agentKeys.keyIssuedFor', { agent: issuedKey.agent_id })}
          </p>
          <p className="mt-1 break-all font-mono text-xs text-lyra-text">{issuedKey.api_key}</p>
        </div>
      )}

      {!keys.length ? (
        <p className="text-sm text-lyra-muted">{t('agentKeys.empty')}</p>
      ) : (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-lyra-borderSoft">
                <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                  {t('agentKeys.agent')}
                </th>
                <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                  {t('agentKeys.created')}
                </th>
                <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                  {t('agentKeys.lastUsed')}
                </th>
                <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                  {t('agentKeys.status')}
                </th>
                <th scope="col" className="pb-2" />
              </tr>
            </thead>
            <tbody>
              {keys.map((k) => (
                <tr key={k.agent_id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-2.5 font-mono text-xs text-lyra-goldBright">{k.agent_id}</td>
                  <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">
                    {new Date(k.created_at).toLocaleDateString()}
                  </td>
                  <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">
                    {k.last_used_at ? new Date(k.last_used_at).toLocaleString() : '—'}
                  </td>
                  <td className="py-2.5">
                    <span className={`font-mono text-xs ${k.revoked ? 'text-lyra-bad' : 'text-lyra-good'}`}>
                      {k.revoked ? t('agentKeys.revoked') : t('agentKeys.active')}
                    </span>
                  </td>
                  <td className="py-2.5 text-right">
                    {canManage && !k.revoked && (
                      <button
                        onClick={() => handleRevoke(k.agent_id)}
                        aria-label={t('agentKeys.revokeLabel', { agent: k.agent_id })}
                        className="font-mono text-xs uppercase tracking-wide text-lyra-bad outline-none hover:underline"
                      >
                        {t('agentKeys.revoke')}
                      </button>
                    )}
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
