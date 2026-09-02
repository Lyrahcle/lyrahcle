import { useEffect } from 'react'
import type { RouteTab } from '../lib/routeTabs'
import { subscribeIncidents, subscribeSatellites, UnauthorizedError } from '../lib/api'

type Entity = Record<string, any>
type Page = {
  items: Entity[]
  total?: number
  limit?: number
  offset?: number
  [key: string]: any
}

type IncidentFilters = {
  status: string
  severity: string
  agentId: string
  satelliteId: string
}

type RefLike<T> = { current: T }

interface DashboardSubscriptionsOptions {
  activeTab: RouteTab
  onSessionExpired: () => void
  setSatellites: (value: Entity[]) => void
  setSatellitesLive: (value: boolean) => void
  setSelectedSatelliteId: (value: string | null) => void
  setIncidents: (value: Page) => void
  incidentsOffsetRef: RefLike<number>
  incidentFiltersRef: RefLike<IncidentFilters>
}

export function useDashboardSubscriptions({
  activeTab,
  onSessionExpired,
  setSatellites,
  setSatellitesLive,
  setSelectedSatelliteId,
  setIncidents,
  incidentsOffsetRef,
  incidentFiltersRef,
}: DashboardSubscriptionsOptions) {
  useEffect(() => {
    if (activeTab !== 'orbital') {
      setSatellitesLive(false)
      setSelectedSatelliteId(null)
      return undefined
    }

    return subscribeSatellites({
      onData: (data) => {
        if (!Array.isArray(data)) return
        setSatellites(data as Entity[])
        setSatellitesLive(true)
      },
      onError: (err) => {
        setSatellitesLive(false)
        if (err instanceof UnauthorizedError) onSessionExpired()
      },
    })
  }, [activeTab, onSessionExpired, setSatellites, setSatellitesLive, setSelectedSatelliteId])

  useEffect(() => {
    if (activeTab !== 'incidents') return undefined

    return subscribeIncidents({
      onData: (data) => {
        const filters = incidentFiltersRef.current
        if (
          incidentsOffsetRef.current === 0 &&
          !filters.status &&
          !filters.severity &&
          !filters.agentId &&
          !filters.satelliteId
        ) {
          setIncidents(data)
        }
      },
      onError: (err) => {
        if (err instanceof UnauthorizedError) onSessionExpired()
      },
    })
  }, [activeTab, incidentFiltersRef, incidentsOffsetRef, onSessionExpired, setIncidents])
}
