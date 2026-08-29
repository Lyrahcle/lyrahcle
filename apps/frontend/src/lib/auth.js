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

export async function login(email, password) {
  const res = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    throw new Error(res.status === 401 ? 'Invalid email or password' : 'Login failed')
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

  // Best-effort: revoke the token server-side so it can't be replayed if
  // leaked. The UI has already logged the user out locally either way, so
  // a failure here (offline, expired token, Redis not configured on the
  // backend) is never surfaced to the user.
  if (token) {
    try {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      })
    } catch {
      // ignore — local logout already happened
    }
  }
}

export function authHeaders() {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

const ROLE_RANK = { viewer: 0, operator: 1, admin: 2 }

export function canWrite() {
  return (ROLE_RANK[getRole()] ?? -1) >= ROLE_RANK.operator
}

export function isAdmin() {
  return getRole() === 'admin'
}
