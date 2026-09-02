import { useCallback, useEffect, useState } from 'react'
import Login from './components/Login.jsx'
import ChangePasswordModal from './components/ChangePasswordModal.jsx'
import ApiStatusBanner from './components/ApiStatusBanner.jsx'
import { isAuthenticated, logout, passwordChangeRequired } from './lib/auth'
import { clearEffectivePermissions, loadEffectivePermissions } from './lib/permissions'
import { ApiError, UnauthorizedError } from './lib/api'
import { useI18n } from './lib/I18nContext'
import { lazyComponent } from './lib/lazyComponent'
import { useHashRoute } from './hooks/useHashRoute'

const Dashboard = lazyComponent(() => import('./components/Dashboard.tsx'))

function App() {
  const { t } = useI18n()
  const [authenticated, setAuthenticated] = useState(isAuthenticated())
  const { activeTab, changeTab } = useHashRoute()
  const [permissionsReady, setPermissionsReady] = useState(false)
  const [authNotice, setAuthNotice] = useState('')
  const [permissionsError, setPermissionsError] = useState<ApiError | null>(null)
  const [permissionsRetry, setPermissionsRetry] = useState(0)
  const mustChangePassword = authenticated && passwordChangeRequired()

  const handleLogout = useCallback(() => {
    clearEffectivePermissions()
    logout()
    setAuthNotice('')
    setAuthenticated(false)
  }, [])

  const handleSessionExpired = useCallback(() => {
    clearEffectivePermissions()
    logout()
    setAuthNotice('session_expired')
    setAuthenticated(false)
  }, [])

  useEffect(() => {
    let cancelled = false
    if (!authenticated || mustChangePassword) {
      clearEffectivePermissions()
      setPermissionsReady(false)
      return undefined
    }
    setPermissionsError(null)
    loadEffectivePermissions()
      .then(() => {
        if (!cancelled) setPermissionsReady(true)
      })
      .catch((error: unknown) => {
        if (cancelled) return
        if (error instanceof UnauthorizedError) {
          handleSessionExpired()
          return
        }
        if (error instanceof ApiError) {
          setPermissionsError(error)
          return
        }
        const normalized = new ApiError(error instanceof Error ? error.message : 'failed to load access context', { cause: error })
        if (typeof error === 'object' && error !== null && 'retryable' in error) {
          normalized.retryable = Boolean((error as { retryable?: unknown }).retryable)
        }
        setPermissionsError(normalized)
      })
    return () => {
      cancelled = true
    }
  }, [authenticated, mustChangePassword, handleSessionExpired, permissionsRetry])

  if (!authenticated) {
    return <Login notice={authNotice} onSuccess={() => { setAuthNotice(''); setPermissionsReady(false); setAuthenticated(true) }} />
  }

  if (mustChangePassword) {
    return <ChangePasswordModal required onClose={() => {}} onSecurityChange={handleLogout} />
  }

  if (!permissionsReady) {
    return (
      <div className="min-h-screen bg-transparent text-lyra-text">
        <ApiStatusBanner
          issue={permissionsError}
          onRetry={permissionsError?.retryable ? () => setPermissionsRetry((value) => value + 1) : undefined}
          onDismiss={() => setPermissionsError(null)}
        />
        <main className="mx-auto flex min-h-[70vh] max-w-3xl items-center justify-center px-6 text-center">
          <div>
            <p className="font-mono text-xs tracking-[0.2em] text-lyra-gold">Lyrahcle</p>
            <p className="mt-3 text-sm text-lyra-muted">{permissionsError ? t('httpErrors.accessContextFailed') : t('httpErrors.loadingAccessContext')}</p>
          </div>
        </main>
      </div>
    )
  }

  return (
    <Dashboard
      onLogout={handleLogout}
      onSessionExpired={handleSessionExpired}
      activeTab={activeTab}
      onChangeTab={changeTab}
    />
  )
}

export default App
