import { useCallback, useEffect, useState } from 'react'
import Login from './components/Login.jsx'
import Dashboard from './components/Dashboard.jsx'
import ChangePasswordModal from './components/ChangePasswordModal.jsx'
import ApiStatusBanner from './components/ApiStatusBanner.jsx'
import { isAuthenticated, logout, passwordChangeRequired } from './lib/auth.js'
import { clearEffectivePermissions, loadEffectivePermissions } from './lib/permissions.js'
import { UnauthorizedError } from './lib/api.js'
import { useI18n } from './lib/I18nContext.jsx'
import { DEFAULT_TAB, getTabFromLocation, setTabInLocation, subscribeTabChanges } from './lib/routeTabs.js'

function App() {
  const { t } = useI18n()
  const [authenticated, setAuthenticated] = useState(isAuthenticated())
  const [activeTab, setActiveTab] = useState(() => getTabFromLocation(DEFAULT_TAB))
  const [permissionsReady, setPermissionsReady] = useState(false)
  const [authNotice, setAuthNotice] = useState('')
  const [permissionsError, setPermissionsError] = useState(null)
  const [permissionsRetry, setPermissionsRetry] = useState(0)
  const mustChangePassword = authenticated && passwordChangeRequired()

  const handleChangeTab = useCallback((tab) => {
    setActiveTab(tab)
    setTabInLocation(tab)
  }, [])

  useEffect(() => {
    setTabInLocation(activeTab, { replace: true })
    return subscribeTabChanges(setActiveTab)
  }, [])

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
      .catch((error) => {
        if (cancelled) return
        if (error instanceof UnauthorizedError) {
          handleSessionExpired()
          return
        }
        setPermissionsError(error)
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
            <p className="font-mono text-xs uppercase tracking-[0.2em] text-lyra-gold">Lyrahcle</p>
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
      onChangeTab={handleChangeTab}
    />
  )
}

export default App
