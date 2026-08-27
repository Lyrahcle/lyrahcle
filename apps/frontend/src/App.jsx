import { useCallback, useState } from 'react'
import Login from './components/Login.jsx'
import Dashboard from './components/Dashboard.jsx'
import { isAuthenticated, logout } from './lib/auth.js'

function App() {
  const [authenticated, setAuthenticated] = useState(isAuthenticated())

  const handleLogout = useCallback(() => {
    logout()
    setAuthenticated(false)
  }, [])

  if (!authenticated) {
    return <Login onSuccess={() => setAuthenticated(true)} />
  }

  return <Dashboard onLogout={handleLogout} onSessionExpired={handleLogout} />
}

export default App
