import { render, screen, waitFor } from '@testing-library/react'
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

  it('fails closed when effective permissions cannot be loaded', async () => {
    auth.isAuthenticated.mockReturnValue(true)
    permissions.loadEffectivePermissions.mockRejectedValue(new Error('rbac unavailable'))
    render(<App />)
    await waitFor(() => expect(auth.logout).toHaveBeenCalled())
    expect(screen.queryByText('DASHBOARD VIEW')).not.toBeInTheDocument()
  })
})
