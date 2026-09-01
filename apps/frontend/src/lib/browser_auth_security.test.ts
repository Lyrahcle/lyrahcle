import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getToken, login, logout } from './auth'

describe('browser auth storage hardening', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    vi.restoreAllMocks()
  })

  it('does not restore bearer credentials from persistent localStorage', () => {
    localStorage.setItem('lyra_token', 'persistent-token')
    localStorage.setItem('lyra_role', 'admin')
    expect(getToken()).toBeNull()
  })

  it('stores a successful login only in sessionStorage and purges legacy persistent auth', async () => {
    localStorage.setItem('lyra_token', 'legacy-token')
    localStorage.setItem('lyra_role', 'admin')
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ token: 'session-token', role: 'viewer' }),
    })

    await login('viewer@example.com', 'password')

    expect(sessionStorage.getItem('lyra_token')).toBe('session-token')
    expect(sessionStorage.getItem('lyra_role')).toBe('viewer')
    expect(localStorage.getItem('lyra_token')).toBeNull()
    expect(localStorage.getItem('lyra_role')).toBeNull()
  })

  it('logout clears both current session auth and any legacy persistent auth', async () => {
    sessionStorage.setItem('lyra_token', 'session-token')
    sessionStorage.setItem('lyra_role', 'viewer')
    localStorage.setItem('lyra_token', 'legacy-token')
    localStorage.setItem('lyra_role', 'admin')
    global.fetch = vi.fn().mockResolvedValue({ ok: true })

    await logout()

    expect(sessionStorage.getItem('lyra_token')).toBeNull()
    expect(sessionStorage.getItem('lyra_role')).toBeNull()
    expect(localStorage.getItem('lyra_token')).toBeNull()
    expect(localStorage.getItem('lyra_role')).toBeNull()
  })
})
