import { useEffect, useState } from 'react'
import { fetchPlatformAccess, UnauthorizedError } from '../lib/api'

type PlatformAccessResult = { platform_admin?: boolean } | null | undefined
type FetchPlatformAccess = () => Promise<PlatformAccessResult>

export async function resolvePlatformAccess(
  onSessionExpired: () => void,
  fetchAccess: FetchPlatformAccess = fetchPlatformAccess,
): Promise<boolean> {
  try {
    const result = await fetchAccess()
    return Boolean(result?.platform_admin)
  } catch (err) {
    if (err instanceof UnauthorizedError) {
      onSessionExpired()
    }
    return false
  }
}

export function usePlatformAccess(onSessionExpired: () => void) {
  const [platformAdmin, setPlatformAdmin] = useState(false)

  useEffect(() => {
    let cancelled = false

    void resolvePlatformAccess(onSessionExpired).then((allowed) => {
      if (!cancelled) setPlatformAdmin(allowed)
    })

    return () => {
      cancelled = true
    }
  }, [onSessionExpired])

  return platformAdmin
}
