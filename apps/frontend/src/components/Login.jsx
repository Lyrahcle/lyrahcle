import { useState } from 'react'
import LogoMark from './LogoMark.jsx'
import OrbitRings from './OrbitRings.jsx'
import { login } from '../lib/auth.js'
import { useI18n } from '../lib/I18nContext.jsx'

export default function Login({ onSuccess, onShowRegister }) {
  const { t } = useI18n()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await login(email, password)
      onSuccess()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden px-6">
      <OrbitRings />

      <div className="relative w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center text-center">
          <LogoMark className="mb-4 h-14 w-14" />
          <h1 className="font-display text-2xl font-semibold text-lyra-goldBright">Lyrahcle</h1>
          <p className="mt-1 font-mono text-xs uppercase tracking-[0.3em] text-lyra-muted">
            {t('header.tagline')}
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="rounded-lg border border-lyra-border bg-lyra-panel/40 p-6"
        >
          <div className="mb-4">
            <label htmlFor="email" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
              {t('common.email')}
            </label>
            <input
              id="email"
              type="text"
              autoComplete="username"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
              required
            />
          </div>

          <div className="mb-5">
            <label htmlFor="password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
              {t('common.password')}
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
              required
            />
          </div>

          {error && <p className="mb-4 text-sm text-lyra-bad">{error}</p>}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
          >
            {loading ? t('auth.signingIn') : t('auth.signIn')}
          </button>
        </form>

        <button
          type="button"
          onClick={onShowRegister}
          className="mt-6 w-full text-center font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-muted outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
        >
          {t('auth.createAnAccount')}
        </button>

        <p className="mt-4 text-center font-mono text-[11px] text-lyra-mutedDim">
          {t('auth.slogan')}
        </p>
      </div>
    </div>
  )
}
