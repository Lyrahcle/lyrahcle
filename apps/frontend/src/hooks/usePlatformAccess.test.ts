import { renderHook, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { UnauthorizedError } from '../lib/api'
import { resolvePlatformAccess, usePlatformAccess } from './usePlatformAccess'

describe('resolvePlatformAccess', () => {
  it('returns platform administration access', async () => {
    const onSessionExpired = vi.fn()
    const fetchAccess = vi.fn().mockResolvedValue({ platform_admin: true })

    await expect(resolvePlatformAccess(onSessionExpired, fetchAccess)).resolves.toBe(true)
    expect(onSessionExpired).not.toHaveBeenCalled()
  })

  it('expires the session on unauthorized access', async () => {
    const onSessionExpired = vi.fn()
    const fetchAccess = vi.fn().mockRejectedValue(new UnauthorizedError())

    await expect(resolvePlatformAccess(onSessionExpired, fetchAccess)).resolves.toBe(false)
    expect(onSessionExpired).toHaveBeenCalledTimes(1)
  })
})

describe('usePlatformAccess', () => {
  it('starts without platform administration access', async () => {
    const onSessionExpired = vi.fn()
    const { result } = renderHook(() => usePlatformAccess(onSessionExpired))

    expect(result.current).toBe(false)
    await waitFor(() => expect(result.current).toBe(false))
  })
})
