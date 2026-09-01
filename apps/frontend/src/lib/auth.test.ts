import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { authHeaders, decodeToken, getRole, getToken, isAuthenticated, login, LoginRateLimitedError, logout, prepareCreationOptions, prepareRequestOptions, WebAuthnRequiredError } from './auth'

function mockResponse(partial: Partial<Response> & Record<string, unknown>): Response {
  return partial as Response
}

function makeToken(payload: Record<string, unknown>) {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.fakesignature`
}

describe('auth client', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
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
    sessionStorage.setItem('lyra_token', token)
    expect(isAuthenticated()).toBe(true)
  })

  it('isAuthenticated is false for an expired token', () => {
    const token = makeToken({ exp: Math.floor(Date.now() / 1000) - 3600 })
    sessionStorage.setItem('lyra_token', token)
    expect(isAuthenticated()).toBe(false)
  })

  it('login stores the token and role on success', async () => {
    vi.mocked(global.fetch).mockResolvedValue(mockResponse({
      ok: true,
      json: () => Promise.resolve({ token: 'a-token', role: 'admin' }),
    }))
    await login('admin', 'admin')
    expect(getToken()).toBe('a-token')
    expect(getRole()).toBe('admin')
  })

  it('login throws on invalid credentials without storing anything', async () => {
    vi.mocked(global.fetch).mockResolvedValue(mockResponse({ ok: false, status: 401 }))
    await expect(login('admin', 'wrong')).rejects.toThrow('Invalid email/username or password')
    expect(getToken()).toBeNull()
  })

  it('login throws LoginRateLimitedError when the backend throttles sign-in', async () => {
    vi.mocked(global.fetch).mockResolvedValue(mockResponse({ ok: false, status: 429 }))
    await expect(login('admin', 'admin')).rejects.toBeInstanceOf(LoginRateLimitedError)
    expect(getToken()).toBeNull()
  })

  it('login requires WebAuthn when the account has a security key', async () => {
    vi.mocked(global.fetch).mockResolvedValue(mockResponse({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: 'webauthn_required' }),
    }))
    await expect(login('admin', 'correct')).rejects.toBeInstanceOf(WebAuthnRequiredError)
    expect(getToken()).toBeNull()
  })

  it('prepares WebAuthn creation and request binary fields', () => {
    const creation = prepareCreationOptions({ publicKey: { challenge: 'AQI', user: { id: 'AwQ' }, excludeCredentials: [{ id: 'BQY', type: 'public-key' }] } })
    expect(Array.from(creation.challenge as Uint8Array)).toEqual([1, 2])
    expect(Array.from((creation.user as unknown as { id: Uint8Array }).id)).toEqual([3, 4])
    expect(Array.from((creation.excludeCredentials![0] as unknown as { id: Uint8Array }).id)).toEqual([5, 6])

    const request = prepareRequestOptions({ publicKey: { challenge: 'Bwg', allowCredentials: [{ id: 'CQo', type: 'public-key' }] } })
    expect(Array.from(request.challenge as Uint8Array)).toEqual([7, 8])
    expect(Array.from((request.allowCredentials![0] as unknown as { id: Uint8Array }).id)).toEqual([9, 10])
  })

  it('login throws TotpRequiredError when the account has 2FA enabled', async () => {
    vi.mocked(global.fetch).mockResolvedValue(mockResponse({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: 'totp_required' }),
    }))
    await expect(login('admin', 'correct')).rejects.toMatchObject({ message: 'totp_required' })
    expect(getToken()).toBeNull()
  })

  it('logout clears the stored token and role', async () => {
    sessionStorage.setItem('lyra_token', 'a-token')
    sessionStorage.setItem('lyra_role', 'admin')
    vi.mocked(global.fetch).mockResolvedValue(mockResponse({ ok: true }))
    await logout()
    expect(getToken()).toBeNull()
    expect(getRole()).toBeNull()
  })

  it('logout revokes the token server-side', async () => {
    sessionStorage.setItem('lyra_token', 'a-token')
    vi.mocked(global.fetch).mockResolvedValue(mockResponse({ ok: true }))
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
    sessionStorage.setItem('lyra_token', 'a-token')
    sessionStorage.setItem('lyra_role', 'admin')
    vi.mocked(global.fetch).mockRejectedValue(new Error('network error'))
    await expect(logout()).resolves.not.toThrow()
    expect(getToken()).toBeNull()
    expect(getRole()).toBeNull()
  })

  it('authHeaders returns an empty object with no token', () => {
    expect(authHeaders()).toEqual({})
  })

  it('authHeaders returns a Bearer header with a token', () => {
    sessionStorage.setItem('lyra_token', 'a-token')
    expect(authHeaders()).toEqual({ Authorization: 'Bearer a-token' })
  })
})
