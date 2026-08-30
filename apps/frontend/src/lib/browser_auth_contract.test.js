import { describe, expect, it } from 'vitest'
import fs from 'node:fs'

describe('browser auth security contract', () => {
  it('keeps JWT auth out of persistent localStorage', () => {
    const src = fs.readFileSync(new URL('./auth.js', import.meta.url), 'utf8')
    expect(src).not.toContain('localStorage.getItem(TOKEN_KEY)')
    expect(src).not.toContain('localStorage.setItem(TOKEN_KEY')
    expect(src).toContain('sessionStorage.getItem(TOKEN_KEY)')
    expect(src).toContain('sessionStorage.setItem(TOKEN_KEY')
  })

  it('does not opt authenticated API requests into ambient cookie credentials', () => {
    const api = fs.readFileSync(new URL('./api.js', import.meta.url), 'utf8')
    const auth = fs.readFileSync(new URL('./auth.js', import.meta.url), 'utf8')
    expect(api).not.toContain("credentials: 'include'")
    expect(api).not.toContain('credentials: "include"')
    expect(auth).toContain('Authorization: `Bearer ${token}`')
  })
})
