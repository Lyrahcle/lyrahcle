import { useCallback, useState } from 'react'
import Login from './components/Login.jsx'
import Register from './components/Register.jsx'
import Dashboard from './components/Dashboard.jsx'
import { isAuthenticated, logout } from './lib/auth.js'

function App() {
  const [authenticated, setAuthenticated] = useState(isAuthenticated())
  const [activeTab, setActiveTab] = useState('orbital')
  const [showRegister, setShowRegister] = useState(false)

  const handleLogout = useCallback(() => {
    logout()
    setAuthenticated(false)
  }, [])

  if (!authenticated) {
    if (showRegister) {
      return (
        <Register
          onSuccess={() => {
            setShowRegister(false)
            setAuthenticated(true)
          }}
          onBackToLogin={() => setShowRegister(false)}
        />
      )
    }
    return <Login onSuccess={() => setAuthenticated(true)} onShowRegister={() => setShowRegister(true)} />
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
