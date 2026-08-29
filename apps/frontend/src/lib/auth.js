const TOKEN_KEY = 'lyra_token'
const ROLE_KEY = 'lyra_role'

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function getRole() {
  return localStorage.getItem(ROLE_KEY)
}

export function isAuthenticated() {
  const token = getToken()
  if (!token) return false
  const payload = decodeToken(token)
  if (!payload || !payload.exp) return false
  return payload.exp * 1000 > Date.now()
}

export function decodeToken(token) {
  try {
    const [, payload] = token.split('.')
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

export class TotpRequiredError extends Error {
  constructor() {
    super('totp_required')
  }
}

export class LoginRateLimitedError extends Error {
  constructor() {
    super('login_rate_limited')
  }
}

export async function login(identifier, password, totpCode) {
  const res = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier, password, totp_code: totpCode }),
  })
  if (!res.ok) {
    if (res.status === 429) throw new LoginRateLimitedError()
    if (res.status === 401) {
      const body = typeof res.json === 'function' ? await res.json().catch(() => null) : null
      if (body?.error === 'totp_required') throw new TotpRequiredError()
      throw new Error(totpCode ? 'Invalid two-factor code' : 'Invalid email/username or password')
    }
    throw new Error('Login failed')
  }
  const data = await res.json()
  localStorage.setItem(TOKEN_KEY, data.token)
  localStorage.setItem(ROLE_KEY, data.role)
  return data
}

export async function logout() {
  const token = getToken()
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(ROLE_KEY)

  if (token) {
    try {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      })
    } catch {}
  }
}

export function authHeaders() {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

const ROLE_RANK = { viewer: 0, auditor: 0, analyst: 1, operator: 2, admin: 3 }

export function canWrite() {
  return (ROLE_RANK[getRole()] ?? -1) >= ROLE_RANK.operator
}

export function canInvestigate() {
  return ['analyst', 'operator', 'admin'].includes(getRole())
}

export function canAudit() {
  return ['auditor', 'admin'].includes(getRole())
}

export function isAdmin() {
  return getRole() === 'admin'
}

export function passwordChangeRequired() {
  return Boolean(decodeToken(getToken())?.password_change_required)
}
