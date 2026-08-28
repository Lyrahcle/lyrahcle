import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  createAgentKey,
  createAsset,
  createProgram,
  createSatellite,
  fetchAgentKeys,
  fetchAssets,
  fetchAuditLog,
  fetchPing,
  fetchPrograms,
  fetchRfReadings,
  fetchSatellites,
  revokeAgentKey,
  UnauthorizedError,
} from './api.js'

function mockFetchOnce(ok, body, status = ok ? 200 : 500) {
  global.fetch = vi.fn().mockResolvedValue({
    ok,
    status,
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(typeof body === 'string' ? body : ''),
  })
}

describe('api client', () => {
  beforeEach(() => {
    global.fetch = vi.fn()
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('fetchAssets sends the auth header and returns the parsed body', async () => {
    localStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, [{ id: '1', kind: 'domain', value: 'example.com' }])
    const data = await fetchAssets()
    expect(fetch).toHaveBeenCalledWith('/api/v1/assets/', {
      headers: { Authorization: 'Bearer test-token' },
    })
    expect(data).toEqual([{ id: '1', kind: 'domain', value: 'example.com' }])
  })

  it('fetchAssets throws when the response is not ok', async () => {
    mockFetchOnce(false, null, 500)
    await expect(fetchAssets()).rejects.toThrow('failed to fetch assets')
  })

  it('fetchAssets throws UnauthorizedError and clears the session on a 401', async () => {
    localStorage.setItem('lyra_token', 'stale-token')
    mockFetchOnce(false, null, 401)
    await expect(fetchAssets()).rejects.toBeInstanceOf(UnauthorizedError)
    expect(localStorage.getItem('lyra_token')).toBeNull()
  })

  it('createAsset posts program_id, kind, and value', async () => {
    mockFetchOnce(true, null)
    await createAsset({ programId: 'p1', kind: 'domain', value: 'x.com' })
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/assets/',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ program_id: 'p1', kind: 'domain', value: 'x.com' }),
      })
    )
  })

  it('createAsset surfaces the backend error message', async () => {
    mockFetchOnce(false, 'kind must be one of: domain, ip, satellite, ground_station, firmware', 400)
    await expect(createAsset({ programId: 'p1', kind: 'bogus', value: 'x' })).rejects.toThrow(
      'kind must be one of'
    )
  })

  it('fetchPrograms and createProgram hit the right endpoints', async () => {
    mockFetchOnce(true, [{ id: 'p1', name: 'Alpha' }])
    await fetchPrograms()
    expect(fetch).toHaveBeenCalledWith('/api/v1/programs/', { headers: {} })

    mockFetchOnce(true, { id: 'p1', name: 'Alpha' })
    await createProgram('Alpha')
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/programs/',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ name: 'Alpha' }) })
    )
  })

  it('createSatellite posts the TLE fields', async () => {
    mockFetchOnce(true, null)
    await createSatellite({ noradId: 25544, name: 'ISS', tleLine1: 'l1', tleLine2: 'l2' })
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/satellites/',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ norad_id: 25544, name: 'ISS', tle_line1: 'l1', tle_line2: 'l2' }),
      })
    )
  })

  it('fetchPing calls the ping endpoint without auth', async () => {
    mockFetchOnce(true, { message: 'pong from lyrahcle' })
    const data = await fetchPing()
    expect(fetch).toHaveBeenCalledWith('/api/v1/ping')
    expect(data.message).toBe('pong from lyrahcle')
  })

  it('fetchPing throws when the backend is unreachable', async () => {
    mockFetchOnce(false, null, 500)
    await expect(fetchPing()).rejects.toThrow('backend unreachable')
  })

  it('fetchRfReadings sends the auth header', async () => {
    localStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, [])
    await fetchRfReadings()
    expect(fetch).toHaveBeenCalledWith('/api/v1/rf/readings', {
      headers: { Authorization: 'Bearer test-token' },
    })
  })

  it('fetchSatellites sends the auth header', async () => {
    localStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, [])
    await fetchSatellites()
    expect(fetch).toHaveBeenCalledWith('/api/v1/satellites/', {
      headers: { Authorization: 'Bearer test-token' },
    })
  })

  it('fetchAgentKeys, createAgentKey, and revokeAgentKey hit the right endpoints', async () => {
    mockFetchOnce(true, [])
    await fetchAgentKeys()
    expect(fetch).toHaveBeenCalledWith('/api/v1/agent-keys/', { headers: {} })

    mockFetchOnce(true, { agent_id: 'sdr-1', api_key: 'lyr_agent_x' })
    await createAgentKey('sdr-1')
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/agent-keys/',
      expect.objectContaining({ method: 'POST', body: JSON.stringify({ agent_id: 'sdr-1' }) })
    )

    mockFetchOnce(true, null)
    await revokeAgentKey('sdr-1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/agent-keys/sdr-1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('fetchAuditLog calls the audit-log endpoint', async () => {
    mockFetchOnce(true, [])
    await fetchAuditLog()
    expect(fetch).toHaveBeenCalledWith('/api/v1/audit-log', { headers: {} })
  })
})
