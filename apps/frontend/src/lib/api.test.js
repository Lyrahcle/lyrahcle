import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fetchAssets, fetchPing, fetchRfReadings, fetchSatellites } from './api.js'

function mockFetchOnce(ok, body) {
  global.fetch = vi.fn().mockResolvedValue({
    ok,
    json: () => Promise.resolve(body),
  })
}

describe('api client', () => {
  beforeEach(() => {
    global.fetch = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('fetchAssets calls the assets endpoint and returns the parsed body', async () => {
    mockFetchOnce(true, [{ id: '1', kind: 'domain', value: 'example.com' }])
    const data = await fetchAssets()
    expect(fetch).toHaveBeenCalledWith('/api/v1/assets')
    expect(data).toEqual([{ id: '1', kind: 'domain', value: 'example.com' }])
  })

  it('fetchAssets throws when the response is not ok', async () => {
    mockFetchOnce(false, null)
    await expect(fetchAssets()).rejects.toThrow('failed to fetch assets')
  })

  it('fetchPing calls the ping endpoint', async () => {
    mockFetchOnce(true, { message: 'pong from lyrahcle' })
    const data = await fetchPing()
    expect(fetch).toHaveBeenCalledWith('/api/v1/ping')
    expect(data.message).toBe('pong from lyrahcle')
  })

  it('fetchPing throws when the backend is unreachable', async () => {
    mockFetchOnce(false, null)
    await expect(fetchPing()).rejects.toThrow('backend unreachable')
  })

  it('fetchRfReadings calls the rf readings endpoint', async () => {
    mockFetchOnce(true, [])
    await fetchRfReadings()
    expect(fetch).toHaveBeenCalledWith('/api/v1/rf/readings')
  })

  it('fetchSatellites calls the satellites endpoint', async () => {
    mockFetchOnce(true, [])
    await fetchSatellites()
    expect(fetch).toHaveBeenCalledWith('/api/v1/satellites/')
  })
})
