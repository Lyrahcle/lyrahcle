import { useCallback, useState } from 'react'
import Login from './components/Login.jsx'
import Dashboard from './components/Dashboard.jsx'
import ChangePasswordModal from './components/ChangePasswordModal.jsx'
import { isAuthenticated, logout, passwordChangeRequired } from './lib/auth.js'

function App() {
  const [authenticated, setAuthenticated] = useState(isAuthenticated())
  const [activeTab, setActiveTab] = useState('orbital')

  const handleLogout = useCallback(() => {
    logout()
    setAuthenticated(false)
  }, [])

  if (!authenticated) {
    return <Login onSuccess={() => setAuthenticated(true)} />
  }

  if (passwordChangeRequired()) {
    return <ChangePasswordModal required onClose={() => {}} onSecurityChange={handleLogout} />
  }

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
