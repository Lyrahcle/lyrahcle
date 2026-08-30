import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import { clearEffectivePermissions, setEffectivePermissions } from '../lib/permissions.js'
import RBACPanel from './RBACPanel.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({
  assignUserRBACRole: vi.fn(),
  createRBACRole: vi.fn(),
  fetchUserRBACRoles: vi.fn(),
  removeUserRBACRole: vi.fn(),
}))

const permissions = [
  { key: 'incidents.read', description: 'Read incidents' },
  { key: 'incidents.manage', description: 'Manage incidents' },
  { key: 'alerts.read', description: 'Read alerts' },
]
const roles = [{ id: 'role-1', name: 'analyst-custom', description: 'Custom analyst', system: false, permissions: [permissions[0]] }]
const users = [{ id: 'user-1', email: 'analyst@example.com', username: 'analyst' }]

describe('RBACPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearEffectivePermissions()
    api.fetchUserRBACRoles.mockResolvedValue([])
  })

  it('is read-only with rbac.read', () => {
    setEffectivePermissions(['rbac.read'])
    render(<I18nProvider><RBACPanel permissions={permissions} roles={roles} users={users} onChanged={() => {}} /></I18nProvider>)
    expect(screen.getByText('analyst-custom')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /create custom role/i })).not.toBeInTheDocument()
  })

  it('creates a custom role with selected capabilities', async () => {
    setEffectivePermissions(['rbac.read', 'rbac.manage'])
    api.createRBACRole.mockResolvedValue({})
    const onChanged = vi.fn().mockResolvedValue()
    render(<I18nProvider><RBACPanel permissions={permissions} roles={roles} users={users} onChanged={onChanged} /></I18nProvider>)

    fireEvent.change(screen.getByLabelText(/role name/i), { target: { value: ' Incident responder ' } })
    fireEvent.click(screen.getByRole('checkbox', { name: /incidents\.manage/i }))
    fireEvent.click(screen.getByRole('button', { name: /create custom role/i }))

    await waitFor(() => expect(api.createRBACRole).toHaveBeenCalledWith('Incident responder', '', ['incidents.manage']))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('assigns and removes roles from a selected user', async () => {
    setEffectivePermissions(['rbac.read', 'rbac.manage'])
    api.fetchUserRBACRoles.mockResolvedValueOnce([]).mockResolvedValueOnce([roles[0]]).mockResolvedValueOnce([roles[0]]).mockResolvedValueOnce([])
    render(<I18nProvider><RBACPanel permissions={permissions} roles={roles} users={users} onChanged={() => {}} /></I18nProvider>)

    fireEvent.change(screen.getByLabelText(/^User$/i), { target: { value: 'user-1' } })
    await waitFor(() => expect(api.fetchUserRBACRoles).toHaveBeenCalledWith('user-1'))

    const roleButton = screen.getByRole('button', { name: 'analyst-custom' })
    fireEvent.click(roleButton)
    await waitFor(() => expect(api.assignUserRBACRole).toHaveBeenCalledWith('user-1', 'role-1'))

    fireEvent.click(roleButton)
    await waitFor(() => expect(api.removeUserRBACRole).toHaveBeenCalledWith('user-1', 'role-1'))
  })
})
