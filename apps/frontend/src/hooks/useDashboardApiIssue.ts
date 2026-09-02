import { useCallback, useEffect, useRef, useState } from 'react'
import { API_ERROR_EVENT, ApiError } from '../lib/api'

export function useDashboardApiIssue() {
  const [apiIssue, setApiIssue] = useState<ApiError | null>(null)
  const [retryNonce, setRetryNonce] = useState(0)
  const rateLimitedUntilRef = useRef(0)

  useEffect(() => {
    function handleApiError(event: Event) {
      const issue = (event as CustomEvent<unknown>).detail
      if (!(issue instanceof ApiError)) return
      if (issue.kind === 'rate_limited' && issue.retryAfterSeconds > 0) {
        rateLimitedUntilRef.current = Date.now() + issue.retryAfterSeconds * 1000
      }
      setApiIssue(issue)
    }

    window.addEventListener(API_ERROR_EVENT, handleApiError)
    return () => window.removeEventListener(API_ERROR_EVENT, handleApiError)
  }, [])

  const retryFailedRequests = useCallback(() => {
    if (Date.now() < rateLimitedUntilRef.current) return false
    setApiIssue(null)
    setRetryNonce((value) => value + 1)
    return true
  }, [])

  return {
    apiIssue,
    setApiIssue,
    retryNonce,
    rateLimitedUntilRef,
    retryFailedRequests,
  }
}
