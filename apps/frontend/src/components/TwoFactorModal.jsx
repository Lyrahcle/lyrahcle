import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { disableTotp, enableTotp, fetchMe, setupTotp } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import { useFocusTrap } from '../lib/useFocusTrap.js'
import { safeOtpauthUrl } from '../lib/safeUrl.js'

export default function TwoFactorModal({ onClose, onSecurityChange }) {
  const { t } = useI18n()
  const dialogRef = useFocusTrap()
  const [loading, setLoading] = useState(true)
  const [enabled, setEnabled] = useState(false)
  const [setup, setSetup] = useState(null)
  const [starting, setStarting] = useState(false)
  const [copied, setCopied] = useState(false)
  const [code, setCode] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [enableError, setEnableError] = useState(null)
  const [enableSuccess, setEnableSuccess] = useState(false)
  const [password, setPassword] = useState('')
  const [disabling, setDisabling] = useState(false)
  const [disableError, setDisableError] = useState(null)
  const [disableSuccess, setDisableSuccess] = useState(false)

  useEffect(() => {
    function handleEscape(event) {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [onClose])

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const me = await fetchMe()
        if (!cancelled) setEnabled(Boolean(me.totp_enabled))
      } catch (err) {
        if (!cancelled) setEnableError(err.message)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    load()
    return () => {
      cancelled = true
    }
  }, [])

  async function handleStart() {
    setEnableError(null)
    setStarting(true)
    try {
      const next = await setupTotp()
      if (next.provider !== 'google_authenticator' || !next.secret) throw new Error(t('twoFactorModal.setupFailed'))
      setSetup(next)
      setCode('')
    } catch (err) {
      setEnableError(err.message)
    } finally {
      setStarting(false)
    }
  }

  async function handleCopy() {
    if (!setup?.secret) return
    try {
      await navigator.clipboard.writeText(setup.secret)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      setCopied(false)
    }
  }

  async function handleConfirm(event) {
    event.preventDefault()
    setEnableError(null)
    setConfirming(true)
    try {
      await enableTotp(code)
      setEnableSuccess(true)
      setEnabled(true)
    } catch (err) {
      setEnableError(err.message)
    } finally {
      setConfirming(false)
    }
  }

  async function handleDisable(event) {
    event.preventDefault()
    setDisableError(null)
    setDisabling(true)
    try {
      await disableTotp(password)
      setDisableSuccess(true)
      setEnabled(false)
    } catch (err) {
      setDisableError(err.message)
    } finally {
      setDisabling(false)
    }
  }

  function continueToSignIn() {
    onSecurityChange()
  }

  return createPortal(
    <div className="fixed inset-0 z-[100] overflow-y-auto bg-black/70 p-4 backdrop-blur-[2px] sm:p-6">
      <div className="flex min-h-full items-center justify-center">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="two-factor-title"
        tabIndex={-1}
        className="w-full max-w-xl overflow-hidden rounded-xl border border-lyra-border bg-lyra-panel p-5 shadow-2xl shadow-black/40 outline-none sm:p-6"
      >
        <div className="mb-5 flex items-center justify-between">
          <h2 id="two-factor-title" className="font-display text-sm font-medium text-lyra-goldBright">
            {t('twoFactorModal.title')}
          </h2>
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
          <p className="text-sm text-lyra-muted">{t('twoFactorModal.loading')}</p>
        ) : enableSuccess || disableSuccess ? (
          <div>
            <p role="status" className="mb-2 text-sm text-lyra-good">
              {enableSuccess ? t('twoFactorModal.enableSuccess') : t('twoFactorModal.disableSuccess')}
            </p>
            <p className="mb-5 text-xs text-lyra-muted">{t('twoFactorModal.sessionsRevoked')}</p>
            <button
              type="button"
              onClick={continueToSignIn}
              className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright"
            >
              {t('twoFactorModal.signInAgain')}
            </button>
          </div>
        ) : enabled ? (
          <form onSubmit={handleDisable}>
            <p className="mb-4 text-sm text-lyra-muted">{t('twoFactorModal.enabledIntro')}</p>
            <div className="mb-5">
              <label htmlFor="totp-disable-password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                {t('twoFactorModal.disablePasswordLabel')}
              </label>
              <input
                id="totp-disable-password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
              />
            </div>
            {disableError && <p role="alert" className="mb-4 text-sm text-lyra-bad">{disableError}</p>}
            <div className="flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className="w-full rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold"
              >
                {t('common.cancel')}
              </button>
              <button
                type="submit"
                disabled={disabling}
                className="w-full rounded border border-lyra-bad px-4 py-2 text-sm font-medium text-lyra-bad outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-bad disabled:opacity-50"
              >
                {disabling ? t('twoFactorModal.disabling') : t('twoFactorModal.disableButton')}
              </button>
            </div>
          </form>
        ) : !setup ? (
          <div>
            <p className="mb-4 text-sm text-lyra-muted">{t('twoFactorModal.disabledIntro')}</p>
            <ol className="mb-5 list-decimal space-y-2 pl-5 text-sm text-lyra-muted">
              <li>{t('twoFactorModal.stepInstall')}</li>
              <li>{t('twoFactorModal.stepAdd')}</li>
              <li>{t('twoFactorModal.stepVerify')}</li>
            </ol>
            {enableError && <p role="alert" className="mb-4 text-sm text-lyra-bad">{enableError}</p>}
            <button
              type="button"
              onClick={handleStart}
              disabled={starting}
              className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
            >
              {starting ? t('twoFactorModal.starting') : t('twoFactorModal.startButton')}
            </button>
          </div>
        ) : (
          <form onSubmit={handleConfirm}>
            <p className="mb-4 text-sm text-lyra-muted">{t('twoFactorModal.googleOnlyIntro')}</p>
            <div className="mb-4 rounded border border-lyra-border bg-lyra-bgSoft p-4">
              <p className="mb-2 text-[11px] uppercase tracking-wider text-lyra-mutedDim">{t('twoFactorModal.accountLabel')}</p>
              <p className="mb-4 break-all font-mono text-sm text-lyra-text">{setup.account_name || 'Lyrahcle'}</p>
              <p className="mb-2 text-[11px] uppercase tracking-wider text-lyra-mutedDim">{t('twoFactorModal.secretLabel')}</p>
              <div className="flex items-start gap-2">
                <p className="min-w-0 flex-1 break-all font-mono text-sm text-lyra-goldBright">{setup.secret}</p>
                <button
                  type="button"
                  onClick={handleCopy}
                  className="shrink-0 rounded border border-lyra-border px-2 py-1 text-xs text-lyra-muted outline-none hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold"
                >
                  {copied ? t('twoFactorModal.copied') : t('twoFactorModal.copy')}
                </button>
              </div>
              <p className="mt-3 text-xs text-lyra-mutedDim">{t('twoFactorModal.timeBasedHint')}</p>
              {safeOtpauthUrl(setup.otpauth_url) && (
                <a
                  href={safeOtpauthUrl(setup.otpauth_url)}
                  className="mt-3 inline-flex rounded border border-lyra-gold/50 px-3 py-1.5 text-xs text-lyra-goldBright outline-none transition-opacity hover:opacity-80 focus-visible:ring-1 focus-visible:ring-lyra-gold"
                >
                  {t('twoFactorModal.openGoogleAuthenticator')}
                </a>
              )}
            </div>
            <div className="mb-5">
              <label htmlFor="totp-code" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                {t('twoFactorModal.codeLabel')}
              </label>
              <input
                id="totp-code"
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                pattern="[0-9]{6}"
                maxLength={6}
                value={code}
                onChange={(event) => setCode(event.target.value.replace(/\D/g, ''))}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-center font-mono text-lg tracking-[0.3em] text-lyra-text outline-none focus:border-lyra-gold"
                required
              />
            </div>
            {enableError && <p role="alert" className="mb-4 text-sm text-lyra-bad">{enableError}</p>}
            <div className="flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className="w-full rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold"
              >
                {t('common.cancel')}
              </button>
              <button
                type="submit"
                disabled={confirming || code.length !== 6}
                className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
              >
                {confirming ? t('twoFactorModal.confirming') : t('twoFactorModal.confirmButton')}
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
