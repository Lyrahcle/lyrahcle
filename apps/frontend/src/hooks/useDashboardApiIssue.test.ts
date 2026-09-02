import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { API_ERROR_EVENT, ApiError } from '../lib/api'
import { useDashboardApiIssue } from './useDashboardApiIssue'

describe('useDashboardApiIssue', () => {
  it('captures API errors and clears them on retry', () => {
    const { result } = renderHook(() => useDashboardApiIssue())
    const error = new ApiError('offline', { kind: 'network' })

    act(() => {
      window.dispatchEvent(new CustomEvent(API_ERROR_EVENT, { detail: error }))
    })
    expect(result.current.apiIssue).toBe(error)

    act(() => {
      expect(result.current.retryFailedRequests()).toBe(true)
    })
    expect(result.current.apiIssue).toBeNull()
    expect(result.current.retryNonce).toBe(1)
  })

  it('blocks manual retry while a rate-limit window is active', () => {
    vi.spyOn(Date, 'now').mockReturnValue(1_000)
    const { result } = renderHook(() => useDashboardApiIssue())
    const error = new ApiError('slow down', { kind: 'rate_limited', retryAfterSeconds: 5 })

    act(() => {
      window.dispatchEvent(new CustomEvent(API_ERROR_EVENT, { detail: error }))
    })

    act(() => {
      expect(result.current.retryFailedRequests()).toBe(false)
    })
    expect(result.current.retryNonce).toBe(0)
    expect(result.current.apiIssue).toBe(error)
    vi.restoreAllMocks()
  })
})
