import { describe, expect, it } from 'vitest'
import authSource from './auth.ts?raw'
import apiSource from './api.ts?raw'

describe('browser auth security contract', () => {
  it('keeps JWT auth out of persistent localStorage', () => {
    expect(authSource).not.toContain('localStorage.getItem(TOKEN_KEY)')
    expect(authSource).not.toContain('localStorage.setItem(TOKEN_KEY')
    expect(authSource).toContain('sessionStorage.getItem(TOKEN_KEY)')
    expect(authSource).toContain('sessionStorage.setItem(TOKEN_KEY')
  })

  it('does not opt authenticated API requests into ambient cookie credentials', () => {
    expect(apiSource).not.toContain("credentials: 'include'")
    expect(apiSource).not.toContain('credentials: "include"')
    expect(authSource).toContain('Authorization: `Bearer ${token}`')
  })
})
