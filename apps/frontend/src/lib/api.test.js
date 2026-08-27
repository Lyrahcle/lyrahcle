import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchAssets, fetchPing, fetchRfReadings, fetchSatellites, UnauthorizedError } from './api.js'

function mockFetchOnce(ok, body, status = ok ? 200 : 500) {
  global.fetch = vi.fn().mockResolvedValue({
    ok,
    status,
    json: () => Promise.resolve(body),
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
    expect(fetch).toHaveBeenCalledWith('/api/v1/assets', {
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
})
