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

export async function fetchAssets() {
  const res = await authedFetch('/api/v1/assets')
  if (!res.ok) throw new Error('failed to fetch assets')
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

export async function fetchSpaceCVEs() {
  const res = await authedFetch('/api/v1/threat-intel/space-cves')
  if (!res.ok) throw new Error('failed to fetch space CVEs')
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
  if (!res.ok) {
    const message = await res.text().catch(() => 'decode failed')
    throw new Error(message.trim() || 'decode failed')
  }
  return res.json()
}

export { UnauthorizedError }
