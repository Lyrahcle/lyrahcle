const TOKEN_KEY = 'lyra_token'
const ROLE_KEY = 'lyra_role'

function purgeLegacyPersistentAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(ROLE_KEY)
}

export function getToken() {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function getRole() {
  return sessionStorage.getItem(ROLE_KEY)
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
      if (body?.error === 'webauthn_required') throw new WebAuthnRequiredError()
      throw new Error(totpCode ? 'Invalid two-factor code' : 'Invalid email/username or password')
    }
    throw new Error('Login failed')
  }
  const data = await res.json()
  purgeLegacyPersistentAuth()
  sessionStorage.setItem(TOKEN_KEY, data.token)
  sessionStorage.setItem(ROLE_KEY, data.role)
  return data
}

export async function logout() {
  const token = getToken()
  purgeLegacyPersistentAuth()
  sessionStorage.removeItem(TOKEN_KEY)
  sessionStorage.removeItem(ROLE_KEY)

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

export class WebAuthnRequiredError extends Error {
  constructor() {
    super('webauthn_required')
  }
}

function decodeBase64Url(value) {
  const padding = '='.repeat((4 - (value.length % 4)) % 4)
  const base64 = (value + padding).replace(/-/g, '+').replace(/_/g, '/')
  const binary = atob(base64)
  return Uint8Array.from(binary, (char) => char.charCodeAt(0))
}

function encodeBase64Url(value) {
  const bytes = new Uint8Array(value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

export function prepareCreationOptions(payload) {
  const source = payload.publicKey ?? payload
  return {
    ...source,
    challenge: decodeBase64Url(source.challenge),
    user: { ...source.user, id: decodeBase64Url(source.user.id) },
    excludeCredentials: (source.excludeCredentials ?? []).map((item) => ({ ...item, id: decodeBase64Url(item.id) })),
  }
}

export function prepareRequestOptions(payload) {
  const source = payload.publicKey ?? payload
  return {
    ...source,
    challenge: decodeBase64Url(source.challenge),
    allowCredentials: (source.allowCredentials ?? []).map((item) => ({ ...item, id: decodeBase64Url(item.id) })),
  }
}

export function serializeCredential(credential) {
  const response = credential.response
  const serialized = {
    id: credential.id,
    rawId: encodeBase64Url(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment ?? undefined,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      clientDataJSON: encodeBase64Url(response.clientDataJSON),
    },
  }
  if ('attestationObject' in response) serialized.response.attestationObject = encodeBase64Url(response.attestationObject)
  if ('authenticatorData' in response) serialized.response.authenticatorData = encodeBase64Url(response.authenticatorData)
  if ('signature' in response) serialized.response.signature = encodeBase64Url(response.signature)
  if ('userHandle' in response && response.userHandle) serialized.response.userHandle = encodeBase64Url(response.userHandle)
  if ('getTransports' in response) serialized.response.transports = response.getTransports()
  return serialized
}

export async function loginWithSecurityKey(identifier, password) {
  if (!window.PublicKeyCredential || !navigator.credentials) throw new Error('Security keys are not supported by this browser')
  const start = await fetch('/api/v1/auth/webauthn/login/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier, password }),
  })
  if (!start.ok) {
    if (start.status === 429) throw new LoginRateLimitedError()
    throw new Error('Unable to start security-key authentication')
  }
  const flow = await start.json()
  const credential = await navigator.credentials.get({ publicKey: prepareRequestOptions(flow.options) })
  if (!credential) throw new Error('Security-key authentication was cancelled')
  const finish = await fetch('/api/v1/auth/webauthn/login/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ flow_id: flow.flow_id, credential: serializeCredential(credential) }),
  })
  if (!finish.ok) throw new Error('Security-key verification failed')
  const data = await finish.json()
  purgeLegacyPersistentAuth()
  sessionStorage.setItem(TOKEN_KEY, data.token)
  sessionStorage.setItem(ROLE_KEY, data.role)
  return data
}
