import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  clearEffectivePermissions,
  effectivePermissions,
  hasPermission,
  loadEffectivePermissions,
  permissionsLoaded,
  setEffectivePermissions,
} from './permissions.js'

describe('effective permissions', () => {
  beforeEach(() => {
    clearEffectivePermissions()
    vi.restoreAllMocks()
  })

  it('defaults closed until permissions are loaded', () => {
    expect(permissionsLoaded()).toBe(false)
    expect(hasPermission('incidents.manage')).toBe(false)
  })

  it('uses exact backend capabilities rather than role rank', () => {
    setEffectivePermissions(['incidents.manage', 'detections.read'])
    expect(permissionsLoaded()).toBe(true)
    expect(hasPermission('incidents.manage')).toBe(true)
    expect(hasPermission('incidents.read')).toBe(false)
  })

  it('loads /rbac/me and caches effective capabilities', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ permissions: ['alerts.manage'] }),
    }))
    await loadEffectivePermissions()
    expect(fetch).toHaveBeenCalledWith('/api/v1/rbac/me', expect.any(Object))
    expect(hasPermission('alerts.manage')).toBe(true)
  })

  it('fails closed when /rbac/me cannot be loaded', async () => {
    setEffectivePermissions(['users.manage'])
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }))
    await expect(loadEffectivePermissions()).rejects.toThrow()
    expect(hasPermission('users.manage')).toBe(false)
    expect(effectivePermissions()).toEqual([])
  })
})
