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
