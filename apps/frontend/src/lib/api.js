import { authHeaders, logout } from './auth.js'

class UnauthorizedError extends Error {}

async function authedFetch(url, options = {}) {
  const res = await fetch(url, { ...options, headers: { ...authHeaders(), ...(options.headers ?? {}) } })
  if (res.status === 401) {
    logout()
    throw new UnauthorizedError('session expired')
  }
  return res
}

async function readErrorText(res, fallback) {
  const message = await res.text().catch(() => fallback)
  return message.trim() || fallback
}

export async function fetchAssets(limit = 50, offset = 0) {
  const res = await authedFetch(`/api/v1/assets/?limit=${limit}&offset=${offset}`)
  if (!res.ok) throw new Error('failed to fetch assets')
  return res.json()
}

export async function createAsset({ programId, kind, value }) {
  const res = await authedFetch('/api/v1/assets/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ program_id: programId, kind, value }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create asset'))
}

export async function fetchPrograms() {
  const res = await authedFetch('/api/v1/programs/')
  if (!res.ok) throw new Error('failed to fetch programs')
  return res.json()
}

export async function createProgram(name) {
  const res = await authedFetch('/api/v1/programs/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create program'))
  return res.json()
}

export async function createUser(email, password, role) {
  const res = await authedFetch('/api/v1/users/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password, role }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create user'))
  return res.json()
}

export async function fetchSystemInfo() {
  const res = await authedFetch('/api/v1/system-info')
  if (!res.ok) throw new Error('failed to fetch system info')
  return res.json()
}

export async function fetchMe() {
  const res = await authedFetch('/api/v1/auth/me')
  if (!res.ok) throw new Error('failed to fetch account info')
  return res.json()
}

export async function setupTotp() {
  const res = await authedFetch('/api/v1/auth/totp/setup', { method: 'POST' })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to start two-factor setup'))
  return res.json()
}

export async function enableTotp(code) {
  const res = await authedFetch('/api/v1/auth/totp/enable', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'invalid code'))
}

export async function disableTotp(password) {
  const res = await authedFetch('/api/v1/auth/totp/disable', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to disable two-factor authentication'))
}

export async function changePassword(currentPassword, newPassword) {
  const res = await authedFetch('/api/v1/auth/change-password', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to change password'))
}


export async function fetchSessions() {
  const res = await authedFetch('/api/v1/auth/sessions/')
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch sessions'))
  return res.json()
}

export async function revokeSession(sessionId) {
  const res = await authedFetch(`/api/v1/auth/sessions/${encodeURIComponent(sessionId)}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to revoke session'))
}

export async function revokeAllSessions() {
  const res = await authedFetch('/api/v1/auth/sessions/', { method: 'DELETE' })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to revoke sessions'))
}

export async function fetchPing() {
  const res = await fetch('/api/v1/ping')
  if (!res.ok) throw new Error('backend unreachable')
  return res.json()
}

export async function fetchRfReadings() {
  const res = await authedFetch('/api/v1/rf/readings')
  if (!res.ok) throw new Error('failed to fetch rf readings')
  return res.json()
}

export async function fetchRfAnomalies() {
  const res = await authedFetch('/api/v1/rf/anomalies')
  if (!res.ok) throw new Error('failed to fetch rf anomalies')
  return res.json()
}

export async function fetchSatellites() {
  const res = await authedFetch('/api/v1/satellites/')
  if (!res.ok) throw new Error('failed to fetch satellites')
  return res.json()
}







export function subscribeSatellites({ onData, onError }) {
  const controller = new AbortController()

  async function run() {
    let res
    try {
      res = await fetch('/api/v1/satellites/stream', {
        headers: authHeaders(),
        signal: controller.signal,
      })
    } catch (err) {
      if (err.name !== 'AbortError') onError?.(err)
      return
    }
    if (!res.ok || !res.body) {
      onError?.(new Error(`stream failed with status ${res.status}`))
      return
    }

    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''

    try {
      while (true) {
        const { value, done } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })

        let boundary
        while ((boundary = buffer.indexOf('\n\n')) !== -1) {
          const chunk = buffer.slice(0, boundary)
          buffer = buffer.slice(boundary + 2)

          const dataLine = chunk.split('\n').find((line) => line.startsWith('data: '))
          if (!dataLine) continue
          try {
            onData?.(JSON.parse(dataLine.slice('data: '.length)))
          } catch {
            
          }
        }
      }
    } catch (err) {
      if (err.name !== 'AbortError') onError?.(err)
    }
  }

  run()
  return () => controller.abort()
}

export async function createSatellite({ noradId, name, tleLine1, tleLine2 }) {
  const res = await authedFetch('/api/v1/satellites/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ norad_id: noradId, name, tle_line1: tleLine1, tle_line2: tleLine2 }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create satellite'))
}

export async function fetchSpaceCVEs(limit = 50, offset = 0, filters = {}) {
  const params = new URLSearchParams({ limit, offset })
  if (filters.severity) params.set('severity', filters.severity)
  if (filters.keyword) params.set('keyword', filters.keyword)
  if (filters.publishedFrom) params.set('published_from', filters.publishedFrom)
  if (filters.publishedTo) params.set('published_to', filters.publishedTo)
  if (filters.sort) params.set('sort', filters.sort)
  if (filters.order) params.set('order', filters.order)

  const res = await authedFetch(`/api/v1/threat-intel/space-cves?${params.toString()}`)
  if (!res.ok) throw new Error('failed to fetch space CVEs')
  return res.json()
}

export async function fetchSpaceCVEKeywords() {
  const res = await authedFetch('/api/v1/threat-intel/space-cves/keywords')
  if (!res.ok) throw new Error('failed to fetch CVE keywords')
  return res.json()
}

export async function fetchAssetAlerts(limit = 50, offset = 0) {
  const res = await authedFetch(`/api/v1/asset-alerts/?limit=${limit}&offset=${offset}`)
  if (!res.ok) throw new Error('failed to fetch asset alerts')
  return res.json()
}

export async function acknowledgeAssetAlert(alertId) {
  const res = await authedFetch(`/api/v1/asset-alerts/${alertId}/acknowledge`, { method: 'POST' })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to acknowledge alert'))
}

export async function fetchAgentKeys() {
  const res = await authedFetch('/api/v1/agent-keys/')
  if (!res.ok) throw new Error('failed to fetch agent keys')
  return res.json()
}

export async function createAgentKey(agentId) {
  const res = await authedFetch('/api/v1/agent-keys/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agent_id: agentId }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create agent key'))
  return res.json()
}

export async function revokeAgentKey(agentId) {
  const res = await authedFetch(`/api/v1/agent-keys/${encodeURIComponent(agentId)}`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to revoke agent key'))
}

export async function fetchAuditLog(limit = 50, offset = 0) {
  const res = await authedFetch(`/api/v1/audit-log?limit=${limit}&offset=${offset}`)
  if (!res.ok) throw new Error('failed to fetch audit log')
  return res.json()
}

export async function fetchUsers() {
  const res = await authedFetch('/api/v1/users/')
  if (!res.ok) throw new Error('failed to fetch users')
  return res.json()
}

export async function updateUserRole(userId, role) {
  const res = await authedFetch(`/api/v1/users/${encodeURIComponent(userId)}/role`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ role }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to update role'))
  return res.json()
}

export async function updateUserStatus(userId, enabled) {
  const res = await authedFetch(`/api/v1/users/${encodeURIComponent(userId)}/status`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to update user status'))
  return res.json()
}

export async function deleteUser(userId) {
  const res = await authedFetch(`/api/v1/users/${encodeURIComponent(userId)}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to delete user'))
}

const DECODE_ENDPOINTS = {
  ax25: { path: '/api/v1/tmtc/decode/ax25', field: 'frame_hex' },
  ccsds: { path: '/api/v1/tmtc/decode/ccsds', field: 'packet_hex' },
  pus: { path: '/api/v1/tmtc/decode/pus', field: 'packet_hex' },
  csp: { path: '/api/v1/tmtc/decode/csp', field: 'packet_hex' },
}

export async function decodeFrame(protocol, hex) {
  const endpoint = DECODE_ENDPOINTS[protocol]
  if (!endpoint) throw new Error(`unknown protocol: ${protocol}`)

  const res = await authedFetch(endpoint.path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ [endpoint.field]: hex }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'decode failed'))
  return res.json()
}

export { UnauthorizedError }

export async function fetchIncidents(limit = 50, offset = 0, filters = {}) {
  const params = new URLSearchParams({ limit, offset })
  if (filters.status) params.set('status', filters.status)
  if (filters.severity) params.set('severity', filters.severity)
  if (filters.agentId) params.set('agent_id', filters.agentId)
  if (filters.satelliteId) params.set('satellite_id', filters.satelliteId)
  if (filters.assignedTo) params.set('assigned_to', filters.assignedTo)
  const res = await authedFetch(`/api/v1/incidents/?${params.toString()}`)
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch incidents'))
  return res.json()
}

export async function fetchIncident(incidentId) {
  const res = await authedFetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}`)
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch incident'))
  return res.json()
}

export async function createIncident({ title, description, severity, satelliteId, agentId, assignedTo }) {
  const res = await authedFetch('/api/v1/incidents/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Idempotency-Key': crypto.randomUUID() },
    body: JSON.stringify({
      title,
      description,
      severity,
      satellite_id: satelliteId || '',
      agent_id: agentId || '',
      assigned_to: assignedTo || '',
    }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create incident'))
  return res.json()
}

export async function updateIncident(incidentId, fields) {
  const body = {}
  if (fields.status !== undefined) body.status = fields.status
  if (fields.severity !== undefined) body.severity = fields.severity
  if (fields.assignedTo !== undefined) body.assigned_to = fields.assignedTo
  const res = await authedFetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to update incident'))
  return res.json()
}

export async function addIncidentNote(incidentId, message) {
  const res = await authedFetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}/notes`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to add incident note'))
}

export async function addIncidentEvidence(incidentId, evidence) {
  const res = await authedFetch(`/api/v1/incidents/${encodeURIComponent(incidentId)}/evidence`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      evidence_type: evidence.type,
      evidence_ref: evidence.reference,
      label: evidence.label,
    }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to add incident evidence'))
  return res.json()
}

export function subscribeIncidents({ onData, onError }) {
  const controller = new AbortController()
  async function run() {
    let res
    try {
      res = await fetch('/api/v1/incidents/stream', { headers: authHeaders(), signal: controller.signal })
    } catch (err) {
      if (err.name !== 'AbortError') onError?.(err)
      return
    }
    if (!res.ok || !res.body) {
      onError?.(new Error(`incident stream failed with status ${res.status}`))
      return
    }
    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    try {
      while (true) {
        const { value, done } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        let boundary
        while ((boundary = buffer.indexOf('\n\n')) !== -1) {
          const chunk = buffer.slice(0, boundary)
          buffer = buffer.slice(boundary + 2)
          const dataLine = chunk.split('\n').find((line) => line.startsWith('data: '))
          if (!dataLine) continue
          try {
            onData?.(JSON.parse(dataLine.slice('data: '.length)))
          } catch {}
        }
      }
    } catch (err) {
      if (err.name !== 'AbortError') onError?.(err)
    }
  }
  run()
  return () => controller.abort()
}

export async function fetchDetectionRules(limit = 50, offset = 0) {
  const res = await authedFetch(`/api/v1/detection-rules/?limit=${limit}&offset=${offset}`)
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch detection rules'))
  return res.json()
}

export async function createDetectionRule({ name, description, metric, operator, threshold, severity, cooldownSeconds, autoIncident }) {
  const res = await authedFetch('/api/v1/detection-rules/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name,
      description,
      metric,
      operator,
      threshold,
      severity,
      cooldown_seconds: cooldownSeconds,
      auto_incident: autoIncident,
    }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create detection rule'))
  return res.json()
}

export async function updateDetectionRule(ruleId, fields) {
  const res = await authedFetch(`/api/v1/detection-rules/${encodeURIComponent(ruleId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(fields),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to update detection rule'))
  return res.json()
}


export async function globalSearch(query, signal) {
  const res = await authedFetch(`/api/v1/search?q=${encodeURIComponent(query)}`, { signal })
  if (!res.ok) throw new Error(await readErrorText(res, 'search failed'))
  return res.json()
}

export async function fetchSecurityKeys() {
  const res = await authedFetch('/api/v1/auth/webauthn/credentials/')
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch security keys'))
  return res.json()
}

export async function registerSecurityKey(label, password) {
  if (!window.PublicKeyCredential || !navigator.credentials) throw new Error('Security keys are not supported by this browser')
  const start = await authedFetch('/api/v1/auth/webauthn/credentials/register/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ label, password }),
  })
  if (!start.ok) throw new Error(await readErrorText(start, 'failed to start security-key registration'))
  const flow = await start.json()
  const { prepareCreationOptions, serializeCredential } = await import('./auth.js')
  const credential = await navigator.credentials.create({ publicKey: prepareCreationOptions(flow.options) })
  if (!credential) throw new Error('Security-key registration was cancelled')
  const finish = await authedFetch('/api/v1/auth/webauthn/credentials/register/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ flow_id: flow.flow_id, credential: serializeCredential(credential) }),
  })
  if (!finish.ok) throw new Error(await readErrorText(finish, 'failed to register security key'))
}

export async function removeSecurityKey(credentialId, password) {
  const res = await authedFetch(`/api/v1/auth/webauthn/credentials/${encodeURIComponent(credentialId)}`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to remove security key'))
}

export async function requestPasswordReset(identifier) {
  const res = await fetch('/api/v1/auth/password-reset/request', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier }),
  })
  if (!res.ok) throw new Error('password recovery request failed')
  return res.json()
}

export async function completePasswordReset(token, newPassword, totpCode) {
  const res = await fetch('/api/v1/auth/password-reset/complete', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token, new_password: newPassword, totp_code: totpCode }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'password reset failed'))
}

export async function fetchAgentIdentities() {
  const res = await authedFetch('/api/v1/agents/')
  if (!res.ok) throw new Error('failed to fetch agent identities')
  return res.json()
}

export async function createAgentEnrollmentToken(agentId) {
  const res = await authedFetch('/api/v1/agents/enrollment-tokens', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agent_id: agentId }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create enrollment token'))
  return res.json()
}

export async function updateAgentIdentityStatus(agentId, status) {
  const res = await authedFetch(`/api/v1/agents/${encodeURIComponent(agentId)}/status`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to update agent status'))
}

export async function updateAgentStation(agentId, latitudeDeg, longitudeDeg, altitudeM) {
  const res = await authedFetch(`/api/v1/agents/${encodeURIComponent(agentId)}/station`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ latitude_deg: latitudeDeg, longitude_deg: longitudeDeg, altitude_m: altitudeM }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to update agent station'))
}

export async function fetchAgentConfig(agentId) {
  const res = await authedFetch(`/api/v1/agents/${encodeURIComponent(agentId)}/config`)
  if (res.status === 204) return null
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch agent config'))
  return res.json()
}

export async function fetchAgentConfigHistory(agentId) {
  const res = await authedFetch(`/api/v1/agents/${encodeURIComponent(agentId)}/config/history`)
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch agent config history'))
  return res.json()
}

export async function applyAgentConfig(agentId, frequencyHz, captureIntervalSeconds, satelliteId = '') {
  const res = await authedFetch(`/api/v1/agents/${encodeURIComponent(agentId)}/config`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ frequency_hz: frequencyHz, capture_interval_seconds: captureIntervalSeconds, satellite_id: satelliteId || null }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to apply agent config'))
  return res.json()
}

export async function rollbackAgentConfig(agentId, version) {
  const res = await authedFetch(`/api/v1/agents/${encodeURIComponent(agentId)}/config/rollback`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ version }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to rollback agent config'))
  return res.json()
}


export async function fetchPlatformAccess() {
  const res = await authedFetch('/api/v1/organizations/me')
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to check platform access'))
  return res.json()
}

export async function fetchOrganizations() {
  const res = await authedFetch('/api/v1/organizations/')
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch organizations'))
  return res.json()
}

export async function createOrganization(slug, name) {
  const res = await authedFetch('/api/v1/organizations/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ slug, name }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create organization'))
  return res.json()
}


export async function fetchRBACPermissions() {
  const res = await authedFetch('/api/v1/rbac/permissions')
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch RBAC permissions'))
  return res.json()
}

export async function fetchRBACRoles() {
  const res = await authedFetch('/api/v1/rbac/roles')
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch RBAC roles'))
  return res.json()
}

export async function createRBACRole(name, description, permissions) {
  const res = await authedFetch('/api/v1/rbac/roles', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, permissions }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create RBAC role'))
  return res.json()
}

export async function fetchUserRBACRoles(userId) {
  const res = await authedFetch(`/api/v1/rbac/users/${encodeURIComponent(userId)}/roles`)
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to fetch user roles'))
  return res.json()
}

export async function assignUserRBACRole(userId, roleId) {
  const res = await authedFetch(`/api/v1/rbac/users/${encodeURIComponent(userId)}/roles/${encodeURIComponent(roleId)}`, { method: 'POST' })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to assign role'))
}

export async function removeUserRBACRole(userId, roleId) {
  const res = await authedFetch(`/api/v1/rbac/users/${encodeURIComponent(userId)}/roles/${encodeURIComponent(roleId)}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to remove role'))
}
