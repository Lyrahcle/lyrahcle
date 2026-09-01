import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import ChangePasswordModal from './ChangePasswordModal.jsx'
import TwoFactorModal from './TwoFactorModal.jsx'
import SessionsModal from './SessionsModal.jsx'
import { I18nProvider } from '../lib/I18nContext'

vi.mock('../lib/api', () => ({
  changePassword: vi.fn(),
  disableTotp: vi.fn(),
  enableTotp: vi.fn(),
  setupTotp: vi.fn(),
  fetchMe: vi.fn().mockResolvedValue({ totp_enabled: false }),
  fetchSessions: vi.fn().mockResolvedValue([]),
  revokeAllSessions: vi.fn(),
  revokeSession: vi.fn(),
}))

function withI18n(node) {
  return <I18nProvider>{node}</I18nProvider>
}

describe('security dialogs viewport layout', () => {
  it('renders password dialog through a viewport portal', () => {
    const host = document.createElement('div')
    document.body.appendChild(host)
    render(withI18n(<ChangePasswordModal onClose={() => {}} onSecurityChange={() => {}} />), { container: host })
    const dialog = screen.getByRole('dialog', { name: /change password|cambiar contraseña/i })
    expect(dialog.parentElement?.parentElement?.parentElement).toBe(document.body)
    host.remove()
  })

  it('renders authenticator dialog through a viewport portal', async () => {
    const host = document.createElement('div')
    document.body.appendChild(host)
    render(withI18n(<TwoFactorModal onClose={() => {}} onSecurityChange={() => {}} />), { container: host })
    const dialog = await screen.findByRole('dialog', { name: /google authenticator/i })
    expect(dialog.parentElement?.parentElement?.parentElement).toBe(document.body)
    host.remove()
  })
  it('renders active sessions dialog through a viewport portal', async () => {
    const host = document.createElement('div')
    document.body.appendChild(host)
    render(withI18n(<SessionsModal onClose={() => {}} onCurrentSessionRevoked={() => {}} />), { container: host })
    const dialog = await screen.findByRole('dialog', { name: /active sessions|sesiones activas/i })
    expect(dialog.parentElement?.parentElement?.parentElement).toBe(document.body)
    host.remove()
  })

})
