import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { changePassword } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import { useFocusTrap } from '../lib/useFocusTrap.js'

export default function ChangePasswordModal({ onClose, onSecurityChange, required = false }) {
  const { t } = useI18n()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)

  const dialogRef = useFocusTrap()

  useEffect(() => {
    function handleEscape(e) {
      if (e.key === 'Escape' && !required) onClose()
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [onClose, required])

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)

    if (newPassword !== confirmPassword) {
      setError(t('changePasswordModal.newPasswordsDoNotMatch'))
      return
    }

    setLoading(true)
    try {
      await changePassword(currentPassword, newPassword)
      setSuccess(true)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return createPortal(
    <div className="fixed inset-0 z-[100] overflow-y-auto bg-black/70 p-4 backdrop-blur-[2px] sm:p-6">
      <div className="flex min-h-full items-center justify-center">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="change-password-title"
        tabIndex={-1}
        className="w-full max-w-md overflow-hidden rounded-xl border border-lyra-border bg-lyra-panel p-5 shadow-2xl shadow-black/40 outline-none sm:p-6"
      >
        <div className="mb-5 flex items-center justify-between">
          <h2 id="change-password-title" className="font-display text-sm font-medium text-lyra-goldBright">
            {t('changePasswordModal.title')}
          </h2>
          {!required && (
            <button
              type="button"
              onClick={onClose}
              aria-label={t('common.close')}
              className="rounded text-lyra-muted outline-none transition-colors hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
            >
              ✕
            </button>
          )}
        </div>

        {success ? (
          <div>
            <p role="status" className="mb-2 text-sm text-lyra-good">
              {t('changePasswordModal.success')}
            </p>
            <p className="mb-5 text-xs text-lyra-muted">{t('changePasswordModal.sessionsRevoked')}</p>
            <button
              type="button"
              onClick={onSecurityChange}
              className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright"
            >
              {t('changePasswordModal.signInAgain')}
            </button>
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className="mb-4">
              <label htmlFor="current-password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                {t('changePasswordModal.currentPassword')}
              </label>
              <input
                id="current-password"
                type="password"
                autoComplete="current-password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
              />
            </div>

            <div className="mb-4">
              <label htmlFor="new-password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                {t('changePasswordModal.newPassword')}
              </label>
              <input
                id="new-password"
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
                minLength={12}
                maxLength={72}
              />
            </div>

            <div className="mb-5">
              <label htmlFor="confirm-new-password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                {t('changePasswordModal.confirmNewPassword')}
              </label>
              <input
                id="confirm-new-password"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
                minLength={12}
                maxLength={72}
              />
            </div>

            {error && (
              <p role="alert" className="mb-4 text-sm text-lyra-bad">
                {error}
              </p>
            )}

            <div className="flex gap-3">
              {!required && (
                <button
                  type="button"
                  onClick={onClose}
                  className="w-full rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold"
                >
                  {t('common.cancel')}
                </button>
              )}
              <button
                type="submit"
                disabled={loading}
                className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
              >
                {loading ? t('common.saving') : t('common.save')}
              </button>
            </div>
          </form>
        )}
      </div>
      </div>
    </div>,
    document.body,
  )
}
