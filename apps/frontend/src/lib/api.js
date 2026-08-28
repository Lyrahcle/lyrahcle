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

export async function fetchAssets() {
  const res = await authedFetch('/api/v1/assets/')
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

export async function fetchSatellites() {
  const res = await authedFetch('/api/v1/satellites/')
  if (!res.ok) throw new Error('failed to fetch satellites')
  return res.json()
}

export async function createSatellite({ noradId, name, tleLine1, tleLine2 }) {
  const res = await authedFetch('/api/v1/satellites/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ norad_id: noradId, name, tle_line1: tleLine1, tle_line2: tleLine2 }),
  })
  if (!res.ok) throw new Error(await readErrorText(res, 'failed to create satellite'))
}

export async function fetchSpaceCVEs() {
  const res = await authedFetch('/api/v1/threat-intel/space-cves')
  if (!res.ok) throw new Error('failed to fetch space CVEs')
  return res.json()
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

export async function fetchAuditLog() {
  const res = await authedFetch('/api/v1/audit-log')
  if (!res.ok) throw new Error('failed to fetch audit log')
  return res.json()
}

const DECODE_ENDPOINTS = {
  ax25: { path: '/api/v1/tmtc/decode/ax25', field: 'frame_hex' },
  ccsds: { path: '/api/v1/tmtc/decode/ccsds', field: 'packet_hex' },
  pus: { path: '/api/v1/tmtc/decode/pus', field: 'packet_hex' },
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
