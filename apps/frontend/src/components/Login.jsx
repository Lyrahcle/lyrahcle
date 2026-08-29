import { useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import LogoMark from './LogoMark.jsx'
import LoginConstellation from './LoginConstellation.jsx'
import { login, TotpRequiredError } from '../lib/auth.js'
import { useI18n } from '../lib/I18nContext.jsx'
import pkg from '../../package.json'

const REPO_URL = 'https://github.com/mlko01/lyrahcle'

export default function Login({ onSuccess }) {
  const { t } = useI18n()
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [totpCode, setTotpCode] = useState('')
  const [needsTotp, setNeedsTotp] = useState(false)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await login(identifier, password, needsTotp ? totpCode : undefined)
      onSuccess()
    } catch (err) {
      if (err instanceof TotpRequiredError) {
        setNeedsTotp(true)
      } else {
        setError(err.message)
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden px-6 py-12">
      <LoginConstellation />

      <div className="relative w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center text-center">
          <LogoMark className="mb-5 h-24 w-24" />
          <h1 className="font-display text-3xl font-semibold text-lyra-goldBright">Lyrahcle</h1>
          <p className="mt-1 font-mono text-xs uppercase tracking-[0.3em] text-lyra-muted">
            {t('header.tagline')}
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-6"
        >
          <div className={needsTotp ? 'hidden' : 'mb-4'}>
            <label htmlFor="identifier" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
              {t('common.emailOrUsername')}
            </label>
            <input
              id="identifier"
              type="text"
              autoComplete="username"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
              required
              disabled={needsTotp}
            />
          </div>

          <div className={needsTotp ? 'hidden' : 'mb-5'}>
            <label htmlFor="password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
              {t('common.password')}
            </label>
            <div className="relative">
              <input
                id="password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 pr-10 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
                disabled={needsTotp}
              />
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                className="absolute inset-y-0 right-0 flex items-center px-3 text-lyra-mutedDim outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
              >
                {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>

          {needsTotp && (
            <div className="mb-5">
              <label htmlFor="totp-code" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                {t('auth.totpCode')}
              </label>
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
              <button
                type="button"
                onClick={() => {
                  setNeedsTotp(false)
                  setTotpCode('')
                  setError(null)
                }}
                className="mt-2 font-mono text-[11px] uppercase tracking-wider text-lyra-mutedDim outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
              >
                {t('auth.backToSignIn')}
              </button>
            </div>
          )}

          {error && (
            <p role="alert" className="mb-4 text-sm text-lyra-bad">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
          >
            {loading ? t('auth.signingIn') : t('auth.signIn')}
          </button>
        </form>

        <p className="mt-6 text-center font-mono text-[11px] text-lyra-mutedDim">
          {t('auth.slogan')}
        </p>
      </div>

      <footer className="relative mt-10 flex flex-col items-center gap-1.5 text-center">
        <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-1 font-mono text-[11px] text-lyra-mutedDim">
          <a
            className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
            href="mailto:manu.ciberseguridad@gmail.com"
          >
            manu.ciberseguridad@gmail.com
          </a>
          <span aria-hidden="true">·</span>
          <a
            className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
            href={`${REPO_URL}/tree/main/docs`}
          >
            {t('footer.documentation')}
          </a>
          <span aria-hidden="true">·</span>
          <span>v{pkg.version}</span>
        </div>
        <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-lyra-mutedDim/80">
          {t('footer.designedBy')}
        </p>
      </footer>
    </div>
  )
}
