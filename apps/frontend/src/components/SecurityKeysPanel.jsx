import { useEffect, useState } from 'react'
import { KeyRound, Plus, Trash2 } from 'lucide-react'
import { fetchSecurityKeys, registerSecurityKey, removeSecurityKey } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'

export default function SecurityKeysPanel({ onSecurityChange }) {
  const { t } = useI18n()
  const [keys, setKeys] = useState([])
  const [label, setLabel] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  async function load() {
    setLoading(true)
    try {
      setKeys(await fetchSecurityKeys())
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  async function handleRegister() {
    setError(null)
    setBusy(true)
    try {
      await registerSecurityKey(label.trim() || t('settings.securityKeyDefaultLabel'), password)
      onSecurityChange?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  async function handleRemove(id) {
    setError(null)
    setBusy(true)
    try {
      await removeSecurityKey(id, password)
      onSecurityChange?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="mt-5 border-t border-lyra-borderSoft pt-5">
      <div className="flex items-center gap-2">
        <KeyRound className="h-4 w-4 text-lyra-gold" />
        <p className="font-mono text-[11px] uppercase tracking-wider text-lyra-gold">{t('settings.securityKeys')}</p>
      </div>
      <p className="mt-2 text-xs text-lyra-mutedDim">{t('settings.securityKeysHint')}</p>
      <div className="mt-3 grid gap-2 sm:grid-cols-2">
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
          placeholder={t('settings.currentPassword')}
          className="rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
        />
        <input
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          maxLength={80}
          placeholder={t('settings.securityKeyLabel')}
          className="min-w-0 rounded border border-lyra-border bg-lyra-bg px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
        />
        <button
          type="button"
          onClick={handleRegister}
          disabled={busy}
          className="flex items-center justify-center gap-1.5 rounded border border-lyra-gold/50 px-3 py-2 text-sm text-lyra-goldBright disabled:opacity-50 sm:col-span-2"
        >
          <Plus className="h-4 w-4" />
          {t('settings.addSecurityKey')}
        </button>
      </div>
      {loading ? (
        <p className="mt-3 text-xs text-lyra-mutedDim">{t('common.loading')}</p>
      ) : keys.length === 0 ? (
        <p className="mt-3 text-xs text-lyra-mutedDim">{t('settings.noSecurityKeys')}</p>
      ) : (
        <div className="mt-3 space-y-2">
          {keys.map((key) => (
            <div key={key.id} className="flex items-center justify-between gap-3 rounded border border-lyra-borderSoft px-3 py-2">
              <div className="min-w-0">
                <p className="truncate text-sm text-lyra-text">{key.label}</p>
                <p className="font-mono text-[10px] text-lyra-mutedDim">{new Date(key.created_at).toLocaleString()}</p>
              </div>
              <button type="button" onClick={() => handleRemove(key.id)} disabled={busy} aria-label={t('settings.removeSecurityKey')} className="text-lyra-mutedDim hover:text-lyra-bad disabled:opacity-50">
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}
      {error && <p role="alert" className="mt-3 text-xs text-lyra-bad">{error}</p>}
    </div>
  )
}
