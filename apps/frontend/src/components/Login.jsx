import { useState } from 'react'
import { Eye, EyeOff, KeyRound } from 'lucide-react'
import LogoMark from './LogoMark.jsx'
import LoginConstellation from './LoginConstellation.jsx'
import {
  login,
  loginWithSecurityKey,
  LoginRateLimitedError,
  TotpRequiredError,
  WebAuthnRequiredError,
} from '../lib/auth.js'
import { completePasswordReset, requestPasswordReset } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import pkg from '../../package.json'

const REPO_URL = 'https://github.com/mlko01/lyrahcle'

export default function Login({ onSuccess, notice = '' }) {
  const { t } = useI18n()
  const resetToken = new URLSearchParams(window.location.search).get('token') ?? ''
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [totpCode, setTotpCode] = useState('')
  const [needsTotp, setNeedsTotp] = useState(false)
  const [needsWebAuthn, setNeedsWebAuthn] = useState(false)
  const [recoveryMode, setRecoveryMode] = useState(Boolean(resetToken) ? 'complete' : null)
  const [recoverySent, setRecoverySent] = useState(false)
  const [newPassword, setNewPassword] = useState('')
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      if (needsWebAuthn) {
        await loginWithSecurityKey(identifier, password)
      } else {
        await login(identifier, password, needsTotp ? totpCode : undefined)
      }
      onSuccess()
    } catch (err) {
      if (err instanceof WebAuthnRequiredError) {
        setNeedsWebAuthn(true)
      } else if (err instanceof TotpRequiredError) {
        setNeedsTotp(true)
      } else if (err instanceof LoginRateLimitedError) {
        setError(t('auth.tooManyAttempts'))
      } else {
        setError(err.message)
      }
    } finally {
      setLoading(false)
    }
  }

  async function handleRecoveryRequest(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await requestPasswordReset(identifier)
      setRecoverySent(true)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function handleRecoveryComplete(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await completePasswordReset(resetToken, newPassword, totpCode)
      window.history.replaceState({}, '', window.location.pathname)
      setRecoveryMode(null)
      setNewPassword('')
      setTotpCode('')
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  function resetSecondFactor() {
    setNeedsTotp(false)
    setNeedsWebAuthn(false)
    setTotpCode('')
    setError(null)
  }

  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-6 py-12">
      <LoginConstellation />

      <div className="relative w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center text-center">
          <LogoMark className="mb-5 h-24 w-24" />
          <h1 className="font-display text-3xl font-semibold text-lyra-goldBright">Lyrahcle</h1>
          <p className="mt-1 font-mono text-xs tracking-[0.3em] text-lyra-muted">{t('header.tagline')}</p>
        </div>

        {recoveryMode === 'request' ? (
          <form onSubmit={handleRecoveryRequest} className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-6">
            <h2 className="mb-2 font-display text-lg text-lyra-text">{t('auth.recoveryTitle')}</h2>
            <p className="mb-5 text-xs text-lyra-mutedDim">{t('auth.recoveryHint')}</p>
            <label htmlFor="recovery-identifier" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">
              {t('common.emailOrUsername')}
            </label>
            <input
              id="recovery-identifier"
              type="text"
              autoComplete="username"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              className="mb-4 w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
              required
            />
            {recoverySent && <p className="mb-4 text-sm text-lyra-good">{t('auth.recoverySent')}</p>}
            {error && <p role="alert" className="mb-4 text-sm text-lyra-bad">{error}</p>}
            <button type="submit" disabled={loading} className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">
              {t('auth.sendRecoveryLink')}
            </button>
            <button type="button" onClick={() => setRecoveryMode(null)} className="mt-3 w-full text-xs text-lyra-mutedDim hover:text-lyra-goldBright">
              {t('auth.backToSignIn')}
            </button>
          </form>
        ) : recoveryMode === 'complete' ? (
          <form onSubmit={handleRecoveryComplete} className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-6">
            <h2 className="mb-2 font-display text-lg text-lyra-text">{t('auth.resetPassword')}</h2>
            <p className="mb-5 text-xs text-lyra-mutedDim">{t('auth.resetPasswordHint')}</p>
            <label htmlFor="reset-password" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">{t('common.password')}</label>
            <input
              id="reset-password"
              type="password"
              autoComplete="new-password"
              minLength={12}
              maxLength={72}
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className="mb-4 w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
              required
            />
            <label htmlFor="reset-totp" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">{t('auth.totpCodeOptional')}</label>
            <input
              id="reset-totp"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              value={totpCode}
              onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, ''))}
              className="mb-4 w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-center font-mono text-lg tracking-[0.3em] text-lyra-text outline-none focus:border-lyra-gold"
            />
            {error && <p role="alert" className="mb-4 text-sm text-lyra-bad">{error}</p>}
            <button type="submit" disabled={loading} className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">
              {t('auth.resetPassword')}
            </button>
          </form>
        ) : (
          <form onSubmit={handleSubmit} className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-6">
            {notice === 'session_expired' && (
              <p role="status" className="mb-4 rounded border border-lyra-warn/30 bg-lyra-warn/10 px-3 py-2 text-sm text-lyra-text">
                {t('auth.sessionExpired')}
              </p>
            )}
            <div className={needsTotp || needsWebAuthn ? 'hidden' : 'mb-4'}>
              <label htmlFor="identifier" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">{t('common.emailOrUsername')}</label>
              <input
                id="identifier"
                type="text"
                autoComplete="username"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
                disabled={needsTotp || needsWebAuthn}
              />
            </div>

            <div className={needsTotp || needsWebAuthn ? 'hidden' : 'mb-5'}>
              <label htmlFor="password" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">{t('common.password')}</label>
              <div className="relative">
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 pr-10 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                  required
                  disabled={needsTotp || needsWebAuthn}
                />
                <button type="button" onClick={() => setShowPassword((v) => !v)} aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')} className="absolute inset-y-0 right-0 flex items-center px-3 text-lyra-mutedDim hover:text-lyra-goldBright">
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {needsTotp && (
              <div className="mb-5">
                <label htmlFor="totp-code" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">{t('auth.totpCode')}</label>
                <input
                  id="totp-code"
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  pattern="[0-9]{6}"
                  maxLength={6}
                  autoFocus
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, ''))}
                  className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-center font-mono text-lg tracking-[0.3em] text-lyra-text outline-none focus:border-lyra-gold"
                  required
                />
              </div>
            )}

            {needsWebAuthn && (
              <div className="mb-5 rounded border border-lyra-gold/30 bg-lyra-gold/5 p-4 text-center">
                <KeyRound className="mx-auto mb-2 h-6 w-6 text-lyra-goldBright" />
                <p className="text-sm text-lyra-text">{t('auth.securityKeyRequired')}</p>
                <p className="mt-1 text-xs text-lyra-mutedDim">{t('auth.securityKeyHint')}</p>
              </div>
            )}

            {(needsTotp || needsWebAuthn) && (
              <button type="button" onClick={resetSecondFactor} className="mb-4 font-mono text-[11px] tracking-wider text-lyra-mutedDim hover:text-lyra-goldBright">
                {t('auth.backToSignIn')}
              </button>
            )}

            {error && <p role="alert" className="mb-4 text-sm text-lyra-bad">{error}</p>}

            <button type="submit" disabled={loading} className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50">
              {loading ? t('auth.signingIn') : needsWebAuthn ? t('auth.useSecurityKey') : t('auth.signIn')}
            </button>
            {!needsTotp && !needsWebAuthn && (
              <button type="button" onClick={() => setRecoveryMode('request')} className="mt-3 w-full text-xs text-lyra-mutedDim hover:text-lyra-goldBright">
                {t('auth.forgotPassword')}
              </button>
            )}
          </form>
        )}

        <p className="mt-6 text-center font-mono text-[11px] text-lyra-mutedDim">{t('auth.slogan')}</p>
      </div>

      <footer className="relative mt-10 flex flex-col items-center gap-1.5 text-center">
        <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 font-mono text-[11px] text-lyra-mutedDim">
          <a className="outline-none transition-colors hover:text-lyra-goldBright" href="mailto:manu.ciberseguridad@gmail.com">manu.ciberseguridad@gmail.com</a>
          <span aria-hidden="true">·</span>
          <a className="outline-none transition-colors hover:text-lyra-goldBright" href={`${REPO_URL}/tree/main/docs`}>{t('footer.documentation')}</a>
          <span aria-hidden="true">·</span>
          <span>v{pkg.version}</span>
        </div>
        <p className="font-mono text-[10px] tracking-[0.2em] text-lyra-mutedDim/80">{t('footer.designedBy')}</p>
      </footer>
    </div>
  )
}
