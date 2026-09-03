import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  applyAgentConfig,
  createAgentKey,
  createAgentEnrollmentToken,
  createAsset,
  updateAsset,
  deleteAsset,
  createIncident,
  createProgram,
  updateOrganization,
  updateProgram,
  deleteProgram,
  createSatellite,
  updateSatellite,
  createOrbitalEvent,
  fetchAgentConfig,
  fetchAgentConfigHistory,
  fetchAgentKeys,
  fetchAgentIdentities,
  fetchAssets,
  fetchIncident,
  fetchIncidents,
  fetchAuditLog,
  fetchPing,
  fetchPrograms,
  fetchRfReadings,
  fetchRfDoppler,
  fetchSessions,
  rollbackAgentConfig,
  revokeSession,
  revokeAllSessions,
  fetchSatellites,
  fetchSatellitePasses,
  fetchSatelliteExternalSources,
  fetchOrbitalEvents,
  updateIncident,
  addIncidentNote,
  revokeAgentKey,
  updateAgentIdentityStatus,
  updateAgentStation,
  UnauthorizedError,
  ApiError,
  API_ERROR_EVENT,
  fetchEffectivePermissions,
  fetchAgentConfigSigningKeys,
  fetchPlatformRateLimits,
  updateAgentConfigSigningRotationPolicy,
  prepareAgentConfigSigningKeyRotation,
  fetchAgentConfigSigningPublicKey,
  activateAgentConfigSigningKey,
  revokeAgentConfigSigningKey,
  acknowledgeAssetAlert,
  subscribeSatellites,
  subscribeIncidents,
} from './api'

function mockFetchOnce(ok, body, status = ok ? 200 : 500, headers = {}) {
  global.fetch = vi.fn().mockResolvedValue({
    ok,
    status,
    headers: { get: (name) => headers[name] ?? headers[name.toLowerCase()] ?? null },
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(typeof body === 'string' ? body : ''),
  })
}

describe('api client', () => {
  beforeEach(() => {
    global.fetch = vi.fn()
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('fetchAssets sends the auth header, pagination params, and returns the parsed body', async () => {
    sessionStorage.setItem('lyra_token', 'test-token')
    const page = { items: [{ id: '1', kind: 'domain', value: 'example.com' }], total: 1, limit: 50, offset: 0 }
    mockFetchOnce(true, page)
    const data = await fetchAssets()
    expect(fetch).toHaveBeenCalledWith('/api/v1/assets/?limit=50&offset=0', {
      headers: { Authorization: 'Bearer test-token' },
    })
    expect(data).toEqual(page)
  })

  it('fetchAssets throws when the response is not ok', async () => {
    mockFetchOnce(false, null, 500)
    await expect(fetchAssets()).rejects.toThrow('failed to fetch assets')
  })

  it('fetchAssets throws UnauthorizedError and clears the session on a 401', async () => {
    sessionStorage.setItem('lyra_token', 'stale-token')
    mockFetchOnce(false, null, 401)
    const error = await fetchAssets().catch((caught) => caught)
    expect(error).toBeInstanceOf(UnauthorizedError)
    expect(error.endpoint).toBe('/api/v1/assets/?limit=50&offset=0')
    expect(localStorage.getItem('lyra_token')).toBeNull()
  })

  it('preserves HTTP status metadata and publishes actionable API errors', async () => {
    const seen: unknown[] = []
    const listener = (event) => seen.push(event.detail)
    window.addEventListener(API_ERROR_EVENT, listener)
    mockFetchOnce(false, 'forbidden', 403, { 'X-Request-ID': 'req-403' })
    let caught
    try {
      await fetchAssets()
    } catch (error) {
      caught = error
    } finally {
      window.removeEventListener(API_ERROR_EVENT, listener)
    }
    expect(caught).toBeInstanceOf(ApiError)
    expect(caught).toMatchObject({ status: 403, kind: 'forbidden', retryable: false, requestId: 'req-403' })
    expect(seen).toHaveLength(1)
    expect(seen[0]).toBe(caught)
  })

  it('captures Retry-After for rate limited responses', async () => {
    mockFetchOnce(false, 'slow down', 429, { 'Retry-After': '12' })
    await expect(fetchAssets()).rejects.toMatchObject({ status: 429, kind: 'rate_limited', retryAfterSeconds: 12, retryable: true })
  })

  it('turns network failures into retryable ApiError instances', async () => {
    global.fetch = vi.fn().mockRejectedValue(new TypeError('network down'))
    await expect(fetchPing()).rejects.toMatchObject({ status: 0, kind: 'network', retryable: true, endpoint: '/api/v1/ping' })
  })

  it('loads effective permissions through the authenticated API boundary', async () => {
    sessionStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, { permissions: ['assets.read'] })
    await expect(fetchEffectivePermissions()).resolves.toEqual({ permissions: ['assets.read'] })
    expect(fetch).toHaveBeenCalledWith('/api/v1/rbac/me', { headers: { Authorization: 'Bearer test-token' } })
  })

  it('createAsset posts program_id, kind, and value', async () => {
    mockFetchOnce(true, null)
    await createAsset({ programId: 'p1', kind: 'domain', value: 'x.com' })
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/assets/',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ program_id: 'p1', kind: 'domain', value: 'x.com', cpe_uri: '' }),
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

  it('asset and program management APIs encode ids and use tenant mutation endpoints', async () => {
    mockFetchOnce(true, { id: 'asset/1', program_id: 'p1', kind: 'ip', value: '198.51.100.42' })
    await updateAsset('asset/1', { programId: 'p1', kind: 'ip', value: '198.51.100.42' })
    expect(fetch).toHaveBeenCalledWith('/api/v1/assets/asset%2F1', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ program_id: 'p1', kind: 'ip', value: '198.51.100.42', cpe_uri: '' }),
    }))

    mockFetchOnce(true, null, 204)
    await deleteAsset('asset/1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/assets/asset%2F1', expect.objectContaining({ method: 'DELETE' }))

    mockFetchOnce(true, { id: 'program/1', name: 'Beta' })
    await updateProgram('program/1', 'Beta')
    expect(fetch).toHaveBeenCalledWith('/api/v1/programs/program%2F1', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ name: 'Beta' }),
    }))

    mockFetchOnce(true, null, 204)
    await deleteProgram('program/1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/programs/program%2F1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('updateOrganization encodes the id and updates platform organization metadata', async () => {
    mockFetchOnce(true, { id: 'org/1', slug: 'mission-b', name: 'Mission B', enabled: true })
    await updateOrganization('org/1', 'mission-b', 'Mission B')
    expect(fetch).toHaveBeenCalledWith('/api/v1/organizations/org%2F1', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ slug: 'mission-b', name: 'Mission B' }),
    }))
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

  it('updateSatellite encodes the stable id and puts all editable fields', async () => {
    const updated = { id: 'sat/1', norad_id: 25544, name: 'ISS (ZARYA)' }
    mockFetchOnce(true, updated)
    await expect(updateSatellite('sat/1', { noradId: 25544, name: 'ISS (ZARYA)', tleLine1: 'l1', tleLine2: 'l2' })).resolves.toEqual(updated)
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/satellites/sat%2F1',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ norad_id: 25544, name: 'ISS (ZARYA)', tle_line1: 'l1', tle_line2: 'l2' }),
      })
    )
  })

  it('fetchSatelliteExternalSources encodes the satellite id', async () => {
    mockFetchOnce(true, { norad_id: 25544, overall_quality: 'good' })
    await fetchSatelliteExternalSources('sat/1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/satellites/sat%2F1/external-sources', { headers: {} })
  })

  it('fetchPing calls the ping endpoint without auth', async () => {
    mockFetchOnce(true, { message: 'pong from lyrahcle' })
    const data = await fetchPing()
    expect(fetch).toHaveBeenCalledWith('/api/v1/ping', {})
    expect(vi.mocked(fetch).mock.calls[0][1]?.headers).toBeUndefined()
    expect(data.message).toBe('pong from lyrahcle')
  })

  it('fetchPing throws when the backend is unreachable', async () => {
    mockFetchOnce(false, null, 500)
    await expect(fetchPing()).rejects.toThrow('backend unreachable')
  })

  it('fetchRfDoppler uses the protected Doppler endpoint', async () => {
    sessionStorage.setItem('lyra_token', 'token-123')
    mockFetchOnce(true, [])
    await fetchRfDoppler()
    expect(global.fetch).toHaveBeenCalledWith('/api/v1/rf/doppler', expect.objectContaining({
      headers: expect.objectContaining({ Authorization: 'Bearer token-123' }),
    }))
  })

  it('fetchRfReadings sends the auth header', async () => {
    sessionStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, [])
    await fetchRfReadings()
    expect(fetch).toHaveBeenCalledWith('/api/v1/rf/readings', {
      headers: { Authorization: 'Bearer test-token' },
    })
  })

  it('fetchSatellites sends the auth header', async () => {
    sessionStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, [])
    await fetchSatellites()
    expect(fetch).toHaveBeenCalledWith('/api/v1/satellites/', {
      headers: { Authorization: 'Bearer test-token' },
    })
  })

  it('orbital operations client exposes pass prediction and orbital-event APIs', async () => {
    mockFetchOnce(true, { passes: [] })
    await fetchSatellitePasses('sat/1', { latitudeDeg: 40.4168, longitudeDeg: -3.7038, altitudeM: 667, hours: 24, minElevationDeg: 10, limit: 10 })
    expect(fetch).toHaveBeenCalledWith(
      '/api/v1/satellites/sat%2F1/passes?latitude_deg=40.4168&longitude_deg=-3.7038&altitude_m=667&hours=24&min_elevation_deg=10&limit=10',
      { headers: {} },
    )

    mockFetchOnce(true, [])
    await fetchOrbitalEvents('sat-1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/satellites/sat-1/orbital-events?hours=168&limit=50', { headers: {} })

    mockFetchOnce(true, { id: 'evt-1' }, 201)
    await createOrbitalEvent('sat-1', { eventType: 'maneuver', eventTime: '2026-08-30T18:00:00Z', source: 'Operator', confidence: 0.9, severity: 'medium' })
    expect(fetch).toHaveBeenCalledWith('/api/v1/satellites/sat-1/orbital-events', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ event_type: 'maneuver', event_time: '2026-08-30T18:00:00Z', source: 'Operator', confidence: 0.9, severity: 'medium', metadata: {} }),
    }))
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

  it('agent remote config client uses signed config endpoints', async () => {
    mockFetchOnce(true, { version: 2, config: { frequency_hz: 437500000, capture_interval_seconds: 10 } })
    await fetchAgentConfig('sdr-1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/sdr-1/config', { headers: {} })

    mockFetchOnce(true, [])
    await fetchAgentConfigHistory('sdr-1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/sdr-1/config/history', { headers: {} })

    mockFetchOnce(true, { version: 3 }, 201)
    await applyAgentConfig('sdr-1', 437500000, 15)
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/sdr-1/config', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ frequency_hz: 437500000, capture_interval_seconds: 15, satellite_id: null }),
    }))

    mockFetchOnce(true, { version: 4 }, 201)
    await rollbackAgentConfig('sdr-1', 2)
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/sdr-1/config/rollback', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ version: 2 }),
    }))
  })

  it('agent identity client uses enrollment and lifecycle endpoints', async () => {
    mockFetchOnce(true, [])
    await fetchAgentIdentities()
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/', { headers: {} })

    mockFetchOnce(true, { agent_id: 'sdr-1', token: 'one-time', expires_at: '2026-08-30T00:00:00Z' }, 201)
    await createAgentEnrollmentToken('sdr-1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/enrollment-tokens', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ agent_id: 'sdr-1' }),
    }))

    mockFetchOnce(true, null, 204)
    await updateAgentIdentityStatus('sdr-1', 'quarantined')
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/sdr-1/status', expect.objectContaining({
      method: 'PATCH',
      body: JSON.stringify({ status: 'quarantined' }),
    }))

    mockFetchOnce(true, null, 204)
    await updateAgentStation('sdr-1', 40.4168, -3.7038, 667)
    expect(fetch).toHaveBeenCalledWith('/api/v1/agents/sdr-1/station', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ latitude_deg: 40.4168, longitude_deg: -3.7038, altitude_m: 667 }),
    }))
  })

  it('fetchAuditLog calls the audit-log endpoint with pagination params', async () => {
    mockFetchOnce(true, { items: [], total: 0, limit: 50, offset: 0 })
    await fetchAuditLog()
    expect(fetch).toHaveBeenCalledWith('/api/v1/audit-log?limit=50&offset=0', { headers: {} })
  })
  it('incident client uses validated API shapes', async () => {
    mockFetchOnce(true, { items: [], total: 0, limit: 50, offset: 0 })
    await fetchIncidents(50, 0, { status: 'new', severity: 'high', agentId: 'sdr-1', satelliteId: 'sat-1' })
    expect(fetch).toHaveBeenCalledWith('/api/v1/incidents/?limit=50&offset=0&status=new&severity=high&agent_id=sdr-1&satellite_id=sat-1', { headers: {} })

    mockFetchOnce(true, { incident: { id: 'i1' }, events: [] })
    await fetchIncident('i1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/incidents/i1', { headers: {} })

    mockFetchOnce(true, { id: 'i1' }, 201)
    await createIncident({ title: 'RF anomaly', description: 'Review signal', severity: 'high', satelliteId: '', agentId: 'sdr-1', assignedTo: '' })
    expect(fetch).toHaveBeenCalledWith('/api/v1/incidents/', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ title: 'RF anomaly', description: 'Review signal', severity: 'high', satellite_id: '', agent_id: 'sdr-1', assigned_to: '' }),
    }))

    mockFetchOnce(true, { id: 'i1', status: 'investigating' })
    await updateIncident('i1', { status: 'investigating' })
    expect(fetch).toHaveBeenCalledWith('/api/v1/incidents/i1', expect.objectContaining({
      method: 'PATCH',
      body: JSON.stringify({ status: 'investigating' }),
    }))

    mockFetchOnce(true, null, 204)
    await addIncidentNote('i1', 'Reviewed')
    expect(fetch).toHaveBeenCalledWith('/api/v1/incidents/i1/notes', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ message: 'Reviewed' }),
    }))
  })

  it('session client lists and revokes only through authenticated session endpoints', async () => {
    sessionStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, [{ id: '11111111-1111-1111-1111-111111111111', current: true }])
    const sessions = await fetchSessions()
    expect(sessions).toHaveLength(1)
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/sessions/', { headers: { Authorization: 'Bearer test-token' } })

    mockFetchOnce(true, null, 204)
    await revokeSession('11111111-1111-1111-1111-111111111111')
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/sessions/11111111-1111-1111-1111-111111111111', expect.objectContaining({ method: 'DELETE' }))

    mockFetchOnce(true, null, 204)
    await revokeAllSessions()
    expect(fetch).toHaveBeenCalledWith('/api/v1/auth/sessions/', expect.objectContaining({ method: 'DELETE' }))
  })



  it('platform rate limit policy is visible through the authenticated platform endpoint', async () => {
    sessionStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, { tenant_read_per_minute: 1200, actor_read_per_minute: 300 })
    const result = await fetchPlatformRateLimits()
    expect(result.tenant_read_per_minute).toBe(1200)
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/platform/rate-limits', { headers: { Authorization: 'Bearer test-token' } })
  })


  it('signing key lifecycle client uses platform key-management endpoints', async () => {
    sessionStorage.setItem('lyra_token', 'test-token')
    mockFetchOnce(true, { provider: 'local-managed', managed: true, keys: [] })
    await fetchAgentConfigSigningKeys()
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/signing-keys/agent-config/', { headers: { Authorization: 'Bearer test-token' } })

    const policy = { automatic_prepare: true, interval_seconds: 7776000, staged_alert_after_seconds: 86400, grace_alert_before_seconds: 43200 }
    mockFetchOnce(true, policy)
    await updateAgentConfigSigningRotationPolicy(policy)
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/signing-keys/agent-config/policy', {
      method: 'PUT',
      headers: { Authorization: 'Bearer test-token', 'Content-Type': 'application/json' },
      body: JSON.stringify(policy),
    })

    mockFetchOnce(true, { key: { kid: 'kid-a', status: 'staged' }, public_key_pem: 'PUBLIC' }, 201)
    await prepareAgentConfigSigningKeyRotation()
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/signing-keys/agent-config/prepare', { method: 'POST', headers: { Authorization: 'Bearer test-token' } })

    mockFetchOnce(true, { kid: 'kid-a', public_key_pem: 'PUBLIC' })
    await fetchAgentConfigSigningPublicKey('kid/a')
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/signing-keys/agent-config/kid%2Fa/public-key', { headers: { Authorization: 'Bearer test-token' } })

    mockFetchOnce(true, { kid: 'kid-a', status: 'active' })
    await activateAgentConfigSigningKey('kid-a', 3600)
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/signing-keys/agent-config/kid-a/activate', {
      method: 'POST',
      headers: { Authorization: 'Bearer test-token', 'Content-Type': 'application/json' },
      body: JSON.stringify({ grace_seconds: 3600 }),
    })

    mockFetchOnce(true, { kid: 'kid-old', status: 'revoked' })
    await revokeAgentConfigSigningKey('kid-old')
    expect(fetch).toHaveBeenLastCalledWith('/api/v1/signing-keys/agent-config/kid-old/revoke', { method: 'POST', headers: { Authorization: 'Bearer test-token' } })
  })

  it('encodes asset-alert identifiers before mutating alerts', async () => {
    mockFetchOnce(true, null, 204)
    await acknowledgeAssetAlert('alert/1')
    expect(fetch).toHaveBeenCalledWith('/api/v1/asset-alerts/alert%2F1/acknowledge', expect.objectContaining({ method: 'POST' }))
  })

  it('shares resilient SSE parsing across satellite and incident streams', async () => {
    const encoder = new TextEncoder()
    const streamResponse = (payload) => {
      const chunks = [
        encoder.encode('event: update\r\ndata:'),
        encoder.encode(` ${JSON.stringify(payload)}\r`),
        encoder.encode('\n\r\n'),
      ]
      let index = 0
      return {
        ok: true,
        status: 200,
        headers: { get: () => null },
        body: {
          getReader: () => ({
            read: async () => index < chunks.length ? { value: chunks[index++], done: false } : { value: undefined, done: true },
            releaseLock: vi.fn(),
          }),
        },
      }
    }

    global.fetch = vi.fn()
      .mockResolvedValueOnce(streamResponse({ id: 'sat-1', name: 'ISS' }))
      .mockResolvedValueOnce(streamResponse({ id: 'incident-1', severity: 'high' }))

    const satellite = await new Promise((resolve, reject) => {
      subscribeSatellites({ onData: resolve, onError: reject })
    })
    const incident = await new Promise((resolve, reject) => {
      subscribeIncidents({ onData: resolve, onError: reject })
    })

    expect(satellite).toEqual({ id: 'sat-1', name: 'ISS' })
    expect(incident).toEqual({ id: 'incident-1', severity: 'high' })
    expect(vi.mocked(fetch).mock.calls[0][0]).toBe('/api/v1/satellites/stream')
    expect(vi.mocked(fetch).mock.calls[1][0]).toBe('/api/v1/incidents/stream')
  })

})
