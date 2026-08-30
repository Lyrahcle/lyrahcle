import { useCallback, useEffect, useState } from 'react'
import Login from './components/Login.jsx'
import Dashboard from './components/Dashboard.jsx'
import ChangePasswordModal from './components/ChangePasswordModal.jsx'
import { isAuthenticated, logout, passwordChangeRequired } from './lib/auth.js'
import { clearEffectivePermissions, loadEffectivePermissions } from './lib/permissions.js'

function App() {
  const [authenticated, setAuthenticated] = useState(isAuthenticated())
  const [activeTab, setActiveTab] = useState('orbital')
  const [permissionsReady, setPermissionsReady] = useState(false)
  const mustChangePassword = authenticated && passwordChangeRequired()

  const handleLogout = useCallback(() => {
    clearEffectivePermissions()
    logout()
    setAuthenticated(false)
  }, [])

  useEffect(() => {
    let cancelled = false
    if (!authenticated || mustChangePassword) {
      clearEffectivePermissions()
      setPermissionsReady(false)
      return undefined
    }
    loadEffectivePermissions()
      .then(() => {
        if (!cancelled) setPermissionsReady(true)
      })
      .catch(() => {
        if (!cancelled) handleLogout()
      })
    return () => {
      cancelled = true
    }
  }, [authenticated, mustChangePassword, handleLogout])

  if (!authenticated) {
    return <Login onSuccess={() => { setPermissionsReady(false); setAuthenticated(true) }} />
  }

  if (mustChangePassword) {
    return <ChangePasswordModal required onClose={() => {}} onSecurityChange={handleLogout} />
  }

  if (!permissionsReady) return null

  return (
    <Dashboard
      onLogout={handleLogout}
      onSessionExpired={handleLogout}
      activeTab={activeTab}
      onChangeTab={setActiveTab}
    />
  )
}

export default App
