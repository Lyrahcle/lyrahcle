import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.jsx'
import * as auth from './lib/auth.js'
import * as permissions from './lib/permissions.js'

vi.mock('./lib/auth.js', () => ({
  isAuthenticated: vi.fn(),
  logout: vi.fn(),
  passwordChangeRequired: vi.fn(),
}))
vi.mock('./lib/permissions.js', () => ({
  clearEffectivePermissions: vi.fn(),
  loadEffectivePermissions: vi.fn(),
}))
vi.mock('./components/Login.jsx', () => ({ default: () => <div>LOGIN VIEW</div> }))
vi.mock('./components/Dashboard.jsx', () => ({ default: () => <div>DASHBOARD VIEW</div> }))
vi.mock('./components/ChangePasswordModal.jsx', () => ({ default: ({ required }) => <div>{required ? 'REQUIRED PASSWORD CHANGE' : 'PASSWORD CHANGE'}</div> }))
vi.mock('./components/ApiStatusBanner.jsx', () => ({ default: ({ issue, onRetry }) => issue ? <div>API ERROR: {issue.message}{onRetry && <button onClick={onRetry}>RETRY ACCESS</button>}</div> : <div>LOADING ACCESS</div> }))
vi.mock('./lib/I18nContext.jsx', () => ({ useI18n: () => ({ t: (key) => key }) }))

describe('App authentication boundary', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    auth.isAuthenticated.mockReturnValue(false)
    auth.passwordChangeRequired.mockReturnValue(false)
    permissions.loadEffectivePermissions.mockResolvedValue([])
  })

  it('shows login when there is no authenticated session', () => {
    render(<App />)
    expect(screen.getByText('LOGIN VIEW')).toBeInTheDocument()
    expect(permissions.loadEffectivePermissions).not.toHaveBeenCalled()
  })

  it('forces required password change before loading RBAC permissions', () => {
    auth.isAuthenticated.mockReturnValue(true)
    auth.passwordChangeRequired.mockReturnValue(true)
    render(<App />)
    expect(screen.getByText('REQUIRED PASSWORD CHANGE')).toBeInTheDocument()
    expect(permissions.loadEffectivePermissions).not.toHaveBeenCalled()
  })

  it('loads effective permissions before exposing the dashboard', async () => {
    auth.isAuthenticated.mockReturnValue(true)
    auth.passwordChangeRequired.mockReturnValue(false)
    let resolvePermissions
    permissions.loadEffectivePermissions.mockReturnValue(new Promise((resolve) => { resolvePermissions = resolve }))
    render(<App />)
    expect(screen.queryByText('DASHBOARD VIEW')).not.toBeInTheDocument()
    resolvePermissions([])
    expect(await screen.findByText('DASHBOARD VIEW')).toBeInTheDocument()
  })

  it('keeps access fail-closed but shows retry instead of a blank screen when RBAC is unavailable', async () => {
    auth.isAuthenticated.mockReturnValue(true)
    const error = Object.assign(new Error('rbac unavailable'), { retryable: true })
    permissions.loadEffectivePermissions.mockRejectedValueOnce(error).mockResolvedValueOnce([])
    render(<App />)
    expect(await screen.findByText(/API ERROR: rbac unavailable/)).toBeInTheDocument()
    expect(auth.logout).not.toHaveBeenCalled()
    expect(screen.queryByText('DASHBOARD VIEW')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'RETRY ACCESS' }))
    expect(await screen.findByText('DASHBOARD VIEW')).toBeInTheDocument()
  })
})
