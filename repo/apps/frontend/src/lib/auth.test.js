import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { authHeaders, decodeToken, getRole, getToken, isAuthenticated, login, logout } from './auth.js'

function makeToken(payload) {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fakesignature`
}

describe('auth client', () => {
  beforeEach(() => {
    localStorage.clear()
    global.fetch = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('decodeToken decodes a well-formed JWT payload', () => {
    const token = makeToken({ user_id: 'abc', role: 'admin', exp: 9999999999 })
    expect(decodeToken(token)).toEqual({ user_id: 'abc', role: 'admin', exp: 9999999999 })
  })

  it('decodeToken returns null for a malformed token', () => {
    expect(decodeToken('not-a-jwt')).toBeNull()
  })

  it('isAuthenticated is false with no token stored', () => {
    expect(isAuthenticated()).toBe(false)
  })

  it('isAuthenticated is true for a token with a future expiry', () => {
    const token = makeToken({ exp: Math.floor(Date.now() / 1000) + 3600 })
    localStorage.setItem('lyra_token', token)
    expect(isAuthenticated()).toBe(true)
  })

  it('isAuthenticated is false for an expired token', () => {
    const token = makeToken({ exp: Math.floor(Date.now() / 1000) - 3600 })
    localStorage.setItem('lyra_token', token)
    expect(isAuthenticated()).toBe(false)
  })

  it('login stores the token and role on success', async () => {
    global.fetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ token: 'a-token', role: 'admin' }),
    })
    await login('admin', 'admin')
    expect(getToken()).toBe('a-token')
    expect(getRole()).toBe('admin')
  })

  it('login throws on invalid credentials without storing anything', async () => {
    global.fetch.mockResolvedValue({ ok: false, status: 401 })
    await expect(login('admin', 'wrong')).rejects.toThrow('Invalid email/username or password')
    expect(getToken()).toBeNull()
  })

  it('login throws TotpRequiredError when the account has 2FA enabled', async () => {
    global.fetch.mockResolvedValue({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: 'totp_required' }),
    })
    await expect(login('admin', 'correct')).rejects.toMatchObject({ message: 'totp_required' })
    expect(getToken()).toBeNull()
  })

  it('logout clears the stored token and role', async () => {
    localStorage.setItem('lyra_token', 'a-token')
    localStorage.setItem('lyra_role', 'admin')
    global.fetch.mockResolvedValue({ ok: true })
    await logout()
    expect(getToken()).toBeNull()
    expect(getRole()).toBeNull()
  })

  it('logout revokes the token server-side', async () => {
    localStorage.setItem('lyra_token', 'a-token')
    global.fetch.mockResolvedValue({ ok: true })
    await logout()
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/auth/logout',
      expect.objectContaining({ method: 'POST', headers: { Authorization: 'Bearer a-token' } }),
    )
  })

  it('logout does not call the backend when there was no token', async () => {
    await logout()
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it('logout clears local state even if the backend call fails', async () => {
    localStorage.setItem('lyra_token', 'a-token')
    localStorage.setItem('lyra_role', 'admin')
    global.fetch.mockRejectedValue(new Error('network error'))
    await expect(logout()).resolves.not.toThrow()
    expect(getToken()).toBeNull()
    expect(getRole()).toBeNull()
  })

  it('authHeaders returns an empty object with no token', () => {
    expect(authHeaders()).toEqual({})
  })

  it('authHeaders returns a Bearer header with a token', () => {
    localStorage.setItem('lyra_token', 'a-token')
    expect(authHeaders()).toEqual({ Authorization: 'Bearer a-token' })
  })
})
