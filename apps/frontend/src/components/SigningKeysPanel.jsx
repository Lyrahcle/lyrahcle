import { useEffect, useMemo, useState } from 'react'
import {
  activateAgentConfigSigningKey,
  fetchAgentConfigSigningKeys,
  fetchAgentConfigSigningPublicKey,
  prepareAgentConfigSigningKeyRotation,
  revokeAgentConfigSigningKey,
  updateAgentConfigSigningRotationPolicy,
} from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import ConfirmDialog from './ConfirmDialog.jsx'
import TableScroll from './TableScroll.jsx'

function shortKID(kid = '') {
  if (kid.length <= 40) return kid
  return `${kid.slice(0, 24)}…${kid.slice(-12)}`
}

function statusClass(status) {
  if (status === 'active') return 'border-lyra-ok/50 text-lyra-ok'
  if (status === 'staged') return 'border-lyra-gold/50 text-lyra-goldBright'
  if (status === 'grace') return 'border-lyra-warn/50 text-lyra-warn'
  return 'border-lyra-bad/50 text-lyra-bad'
}

function policyDraftFromState(result) {
  const policy = result?.rotation_policy
  return {
    automaticPrepare: Boolean(policy?.automatic_prepare),
    intervalDays: String(Math.round((policy?.interval_seconds || 7776000) / 86400)),
    stagedAlertHours: String(Math.round((policy?.staged_alert_after_seconds || 86400) / 3600)),
    graceAlertHours: String(Math.round((policy?.grace_alert_before_seconds || 86400) / 3600)),
  }
}

function warningLabel(t, warning) {
  const key = `signingKeys.warning_${warning?.code || 'unknown'}`
  return t(key)
}

export default function SigningKeysPanel() {
  const { t } = useI18n()
  const [state, setState] = useState(null)
  const [error, setError] = useState(null)
  const [busy, setBusy] = useState(false)
  const [pendingAction, setPendingAction] = useState(null)
  const [publicKey, setPublicKey] = useState(null)
  const [graceHours, setGraceHours] = useState('168')
  const [policyDraft, setPolicyDraft] = useState(policyDraftFromState(null))

  function applyState(result) {
    setState(result)
    if (result?.grace_period_seconds) setGraceHours(String(Math.round(result.grace_period_seconds / 3600)))
    if (result?.rotation_policy) setPolicyDraft(policyDraftFromState(result))
  }

  async function reload() {
    const result = await fetchAgentConfigSigningKeys()
    applyState(result)
  }

  useEffect(() => {
    let cancelled = false
    fetchAgentConfigSigningKeys()
      .then((result) => {
        if (!cancelled) applyState(result)
      })
      .catch((err) => { if (!cancelled) setError(err.message) })
    return () => { cancelled = true }
  }, [])

  const staged = useMemo(() => state?.keys?.find((key) => key.status === 'staged') || null, [state])

  async function savePolicy(event) {
    event.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await updateAgentConfigSigningRotationPolicy({
        automatic_prepare: policyDraft.automaticPrepare,
        interval_seconds: Number(policyDraft.intervalDays) * 86400,
        staged_alert_after_seconds: Number(policyDraft.stagedAlertHours) * 3600,
        grace_alert_before_seconds: Number(policyDraft.graceAlertHours) * 3600,
      })
      await reload()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function prepare() {
    setBusy(true)
    setError(null)
    try {
      const result = await prepareAgentConfigSigningKeyRotation()
      setPublicKey({ kid: result.key.kid, pem: result.public_key_pem })
      await reload()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function showPublicKey(kid) {
    setBusy(true)
    setError(null)
    try {
      const result = await fetchAgentConfigSigningPublicKey(kid)
      setPublicKey({ kid: result.kid, pem: result.public_key_pem })
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function confirmPendingAction() {
    if (!pendingAction) return
    const { type, kid } = pendingAction
    setBusy(true)
    setError(null)
    try {
      if (type === 'activate') {
        await activateAgentConfigSigningKey(kid, Number(graceHours) * 3600)
      } else {
        await revokeAgentConfigSigningKey(kid)
      }
      setPublicKey(null)
      setPendingAction(null)
      await reload()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  if (!state && !error) return <p className="text-sm text-lyra-muted">{t('common.loading')}</p>

  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p className="text-sm text-lyra-muted">{t('signingKeys.hint')}</p>
          <p className="mt-1 font-mono text-[10px] uppercase text-lyra-mutedDim">
            {t('signingKeys.provider')}: {state?.provider || '—'}
          </p>
        </div>
        <button
          type="button"
          disabled={busy || !state?.managed || Boolean(staged)}
          onClick={prepare}
          className="rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50"
        >
          {t('signingKeys.prepare')}
        </button>
      </div>

      {state?.provider_info?.external && (
        <section aria-label="External signing custody" className="rounded border border-lyra-ok/30 bg-lyra-ok/5 p-3 text-xs text-lyra-text">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="font-mono uppercase text-lyra-ok">External key custody</span>
            <span className={state?.provider_health?.healthy ? 'text-lyra-ok' : 'text-lyra-bad'}>
              {state?.provider_health?.healthy ? 'Signer healthy' : 'Signer unavailable'}
            </span>
          </div>
          <p className="mt-2 break-all font-mono text-[10px] text-lyra-muted">{state.provider_info.endpoint}</p>
          <p className="mt-1 font-mono text-[10px] text-lyra-muted">KID: {state.provider_info.active_kid || '—'} · {state.provider_info.algorithm || '—'}</p>
          {state?.provider_health?.error && <p className="mt-2 text-lyra-bad">{state.provider_health.error}</p>}
        </section>
      )}
      {!state?.managed && state?.provider !== 'disabled' && !state?.provider_info?.external && (
        <p className="rounded border border-lyra-warn/30 bg-lyra-warn/5 p-3 text-xs text-lyra-warn">{t('signingKeys.legacyHint')}</p>
      )}
      {state?.provider === 'disabled' && (
        <p className="rounded border border-lyra-borderSoft p-3 text-xs text-lyra-muted">{t('signingKeys.disabled')}</p>
      )}
      {error && <p role="alert" className="text-sm text-lyra-bad">{error}</p>}

      {state?.managed && state?.rotation_policy && (
        <form onSubmit={savePolicy} className="grid gap-3 rounded border border-lyra-borderSoft bg-lyra-bgSoft/20 p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 className="font-mono text-xs uppercase tracking-wider text-lyra-goldBright">{t('signingKeys.rotationPolicy')}</h3>
              <p className="mt-1 text-xs text-lyra-muted">{t('signingKeys.rotationPolicyHint')}</p>
            </div>
            <button type="submit" disabled={busy} className="rounded border border-lyra-gold/50 px-3 py-1.5 text-xs text-lyra-goldBright disabled:opacity-50">{t('signingKeys.savePolicy')}</button>
          </div>
          <label className="flex items-center gap-2 text-sm text-lyra-text">
            <input type="checkbox" checked={policyDraft.automaticPrepare} onChange={(e) => setPolicyDraft((current) => ({ ...current, automaticPrepare: e.target.checked }))} />
            {t('signingKeys.automaticPrepare')}
          </label>
          <div className="grid gap-3 md:grid-cols-3">
            <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
              {t('signingKeys.intervalDays')}
              <input aria-label={t('signingKeys.intervalDays')} type="number" min="1" max="365" value={policyDraft.intervalDays} onChange={(e) => setPolicyDraft((current) => ({ ...current, intervalDays: e.target.value }))} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm normal-case text-lyra-text" />
            </label>
            <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
              {t('signingKeys.stagedAlertHours')}
              <input aria-label={t('signingKeys.stagedAlertHours')} type="number" min="1" max="720" value={policyDraft.stagedAlertHours} onChange={(e) => setPolicyDraft((current) => ({ ...current, stagedAlertHours: e.target.value }))} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm normal-case text-lyra-text" />
            </label>
            <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
              {t('signingKeys.graceAlertHours')}
              <input aria-label={t('signingKeys.graceAlertHours')} type="number" min="1" max="720" value={policyDraft.graceAlertHours} onChange={(e) => setPolicyDraft((current) => ({ ...current, graceAlertHours: e.target.value }))} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm normal-case text-lyra-text" />
            </label>
          </div>
          <div className="grid gap-2 text-xs text-lyra-muted md:grid-cols-2">
            <p>{t('signingKeys.nextRotation')}: {state.health?.rotation_due ? t('signingKeys.dueNow') : state.health?.next_rotation_at ? new Date(state.health.next_rotation_at).toLocaleString() : '—'}</p>
            <p>{t('signingKeys.automationSafety')}</p>
          </div>
        </form>
      )}

      {state?.health?.warnings?.length > 0 && (
        <section aria-label={t('signingKeys.alerts')} className="grid gap-2">
          <h3 className="font-mono text-xs uppercase tracking-wider text-lyra-warn">{t('signingKeys.alerts')}</h3>
          {state.health.warnings.map((warning, index) => (
            <div key={`${warning.code}-${warning.kid || index}`} className={`rounded border p-3 text-xs ${warning.severity === 'critical' ? 'border-lyra-bad/40 bg-lyra-bad/5 text-lyra-bad' : 'border-lyra-warn/40 bg-lyra-warn/5 text-lyra-warn'}`}>
              <span>{warningLabel(t, warning)}</span>
              {warning.kid && <span className="ml-2 font-mono" title={warning.kid}>{shortKID(warning.kid)}</span>}
              {warning.deadline && <span className="ml-2 text-lyra-muted">{new Date(warning.deadline).toLocaleString()}</span>}
            </div>
          ))}
        </section>
      )}

      {state?.keys?.length > 0 && (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-lyra-borderSoft">
                <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">KID</th>
                <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('signingKeys.status')}</th>
                <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('signingKeys.created')}</th>
                <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('signingKeys.graceUntil')}</th>
                <th className="pb-2" />
              </tr>
            </thead>
            <tbody>
              {state.keys.map((key) => (
                <tr key={key.kid} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-3 pr-4 font-mono text-[10px] text-lyra-goldBright" title={key.kid}>{shortKID(key.kid)}</td>
                  <td className="py-3 pr-4"><span className={`rounded border px-2 py-1 font-mono text-[10px] uppercase ${statusClass(key.status)}`}>{key.status}</span></td>
                  <td className="py-3 pr-4 text-xs text-lyra-muted">{key.created_at ? new Date(key.created_at).toLocaleString() : '—'}</td>
                  <td className="py-3 pr-4 text-xs text-lyra-muted">{key.grace_expires_at ? new Date(key.grace_expires_at).toLocaleString() : '—'}</td>
                  <td className="py-3 text-right">
                    <div className="flex flex-wrap justify-end gap-2">
                      {state.managed && <button type="button" disabled={busy} onClick={() => showPublicKey(key.kid)} className="font-mono text-xs text-lyra-goldBright hover:underline disabled:opacity-50">{t('signingKeys.publicKey')}</button>}
                      {key.status === 'staged' && (
                        <button type="button" disabled={busy} onClick={() => setPendingAction({ type: 'activate', kid: key.kid })} className="font-mono text-xs text-lyra-ok hover:underline disabled:opacity-50">{t('signingKeys.activate')}</button>
                      )}
                      {(key.status === 'staged' || key.status === 'grace') && (
                        <button type="button" disabled={busy} onClick={() => setPendingAction({ type: 'revoke', kid: key.kid })} className="font-mono text-xs text-lyra-bad hover:underline disabled:opacity-50">{t('signingKeys.revoke')}</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}

      {staged && (
        <label className="grid max-w-xs gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
          {t('signingKeys.graceHours')}
          <input type="number" min="1" max="720" value={graceHours} onChange={(e) => setGraceHours(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm normal-case text-lyra-text" />
        </label>
      )}

      {publicKey && (
        <section aria-label={t('signingKeys.publicKey')} className="rounded border border-lyra-gold/30 bg-lyra-gold/5 p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <h3 className="font-mono text-xs uppercase tracking-wider text-lyra-gold">{t('signingKeys.publicKey')}</h3>
              <p className="mt-1 break-all font-mono text-[10px] text-lyra-muted">{publicKey.kid}</p>
            </div>
            <button type="button" onClick={() => setPublicKey(null)} className="text-xs text-lyra-muted hover:text-lyra-text">{t('common.close')}</button>
          </div>
          <p className="mt-3 text-xs text-lyra-muted">{t('signingKeys.distributeHint')}</p>
          <pre className="mt-3 overflow-x-auto whitespace-pre-wrap break-all rounded border border-lyra-borderSoft bg-lyra-bg p-3 text-[10px] text-lyra-text">{publicKey.pem}</pre>
        </section>
      )}
      {pendingAction && (
        <ConfirmDialog
          message={pendingAction.type === 'activate' ? t('signingKeys.activateConfirm') : t('signingKeys.revokeConfirm')}
          confirmLabel={pendingAction.type === 'activate' ? t('signingKeys.activate') : t('signingKeys.revoke')}
          danger={pendingAction.type === 'revoke'}
          busy={busy}
          onConfirm={confirmPendingAction}
          onCancel={() => setPendingAction(null)}
        />
      )}
    </div>
  )
}
