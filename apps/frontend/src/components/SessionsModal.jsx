import { useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { fetchSessions, revokeAllSessions, revokeSession } from '../lib/api'
import { useI18n } from '../lib/I18nContext'
import { useFocusTrap } from '../lib/useFocusTrap'

function formatDate(value) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString()
}

function sessionLabel(userAgent) {
  if (!userAgent) return 'Unknown client'
  const normalized = userAgent.toLowerCase()
  if (normalized.includes('firefox')) return 'Firefox'
  if (normalized.includes('edg/')) return 'Microsoft Edge'
  if (normalized.includes('chrome')) return 'Chrome'
  if (normalized.includes('safari')) return 'Safari'
  return userAgent.slice(0, 48)
}

export default function SessionsModal({ onClose, onCurrentSessionRevoked }) {
  const { t } = useI18n()
  const dialogRef = useFocusTrap(true, onClose)
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [busyId, setBusyId] = useState(null)
  const [revokingAll, setRevokingAll] = useState(false)


  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const data = await fetchSessions()
        if (!cancelled) setSessions(Array.isArray(data) ? data : [])
      } catch (err) {
        if (!cancelled) setError(err.message)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => {
      cancelled = true
    }
  }, [])

  const ordered = useMemo(
    () => [...sessions].sort((a, b) => Number(Boolean(b.current)) - Number(Boolean(a.current)) || new Date(b.created_at) - new Date(a.created_at)),
    [sessions],
  )

  async function handleRevoke(session) {
    setError(null)
    setBusyId(session.id)
    try {
      await revokeSession(session.id)
      if (session.current) {
        onCurrentSessionRevoked()
        return
      }
      setSessions((current) => current.filter((item) => item.id !== session.id))
    } catch (err) {
      setError(err.message)
    } finally {
      setBusyId(null)
    }
  }

  async function handleRevokeAll() {
    setError(null)
    setRevokingAll(true)
    try {
      await revokeAllSessions()
      onCurrentSessionRevoked()
    } catch (err) {
      setError(err.message)
      setRevokingAll(false)
    }
  }

  return createPortal(
    <div className="fixed inset-0 z-[100] overflow-y-auto bg-black/70 p-4 backdrop-blur-[2px] sm:p-6">
      <div className="flex min-h-full items-center justify-center">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="sessions-title"
        tabIndex={-1}
        className="w-full max-w-2xl overflow-hidden rounded-xl border border-lyra-border bg-lyra-panel p-5 shadow-2xl shadow-black/40 outline-none sm:p-6"
      >
        <div className="mb-5 flex items-start justify-between gap-4">
          <div>
            <h2 id="sessions-title" className="font-display text-sm font-medium text-lyra-goldBright">
              {t('sessionsModal.title')}
            </h2>
            <p className="mt-1 text-xs text-lyra-muted">{t('sessionsModal.intro')}</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('common.close')}
            className="rounded text-lyra-muted outline-none transition-colors hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
          >
            ✕
          </button>
        </div>

        {loading ? (
          <p className="text-sm text-lyra-muted">{t('common.loading')}</p>
        ) : (
          <div className="space-y-3">
            {ordered.length === 0 && <p className="text-sm text-lyra-muted">{t('sessionsModal.empty')}</p>}
            {ordered.map((session) => (
              <div key={session.id} className="rounded border border-lyra-border bg-lyra-bgSoft p-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs text-lyra-text">{sessionLabel(session.user_agent)}</span>
                      {session.current && (
                        <span className="rounded border border-lyra-good/50 px-1.5 py-0.5 text-[10px] tracking-wider text-lyra-good">
                          {t('sessionsModal.current')}
                        </span>
                      )}
                    </div>
                    <dl className="mt-2 grid gap-x-5 gap-y-1 text-xs text-lyra-muted sm:grid-cols-2">
                      <div><dt className="inline text-lyra-mutedDim">{t('sessionsModal.ip')}: </dt><dd className="inline font-mono">{session.ip || '—'}</dd></div>
                      <div><dt className="inline text-lyra-mutedDim">{t('sessionsModal.created')}: </dt><dd className="inline">{formatDate(session.created_at)}</dd></div>
                      <div><dt className="inline text-lyra-mutedDim">{t('sessionsModal.expires')}: </dt><dd className="inline">{formatDate(session.expires_at)}</dd></div>
                    </dl>
                  </div>
                  <button
                    type="button"
                    disabled={busyId === session.id}
                    onClick={() => handleRevoke(session)}
                    className="shrink-0 rounded border border-lyra-bad/60 px-3 py-1.5 text-xs text-lyra-bad outline-none transition-opacity hover:opacity-80 focus-visible:ring-1 focus-visible:ring-lyra-bad disabled:opacity-50"
                  >
                    {busyId === session.id ? t('sessionsModal.revoking') : t('sessionsModal.revoke')}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {error && <p role="alert" className="mt-4 text-sm text-lyra-bad">{error}</p>}

        <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={onClose}
            className="rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold"
          >
            {t('common.close')}
          </button>
          <button
            type="button"
            disabled={revokingAll || loading || ordered.length === 0}
            onClick={handleRevokeAll}
            className="rounded border border-lyra-bad px-4 py-2 text-sm font-medium text-lyra-bad outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-bad disabled:opacity-50"
          >
            {revokingAll ? t('sessionsModal.revokingAll') : t('sessionsModal.revokeAll')}
          </button>
        </div>
      </div>
      </div>
    </div>,
    document.body,
  )
}
