import { useEffect, useState } from 'react'
import { fetchOrbitalEvents, fetchSatellitePasses } from '../lib/api'
import type { OrbitalEvent, SatellitePass } from '../lib/apiTypes'
import type { StationCoordinates } from '../lib/groundStations'

export interface MissionOrbitalContext {
  orbitalEvents: OrbitalEvent[] | null
  passes: SatellitePass[] | null
}

export function useMissionOrbitalContext(
  satelliteId: string | null | undefined,
  coordinates: StationCoordinates | null,
  enabled: boolean,
): MissionOrbitalContext {
  const [orbitalEvents, setOrbitalEvents] = useState<OrbitalEvent[] | null>(null)
  const [passes, setPasses] = useState<SatellitePass[] | null>(null)

  useEffect(() => {
    let cancelled = false
    if (!satelliteId || !enabled) {
      setOrbitalEvents([])
      return undefined
    }

    setOrbitalEvents(null)
    fetchOrbitalEvents(satelliteId, 168, 50)
      .then((data) => {
        if (!cancelled) setOrbitalEvents(Array.isArray(data) ? data : [])
      })
      .catch(() => {
        if (!cancelled) setOrbitalEvents([])
      })

    return () => { cancelled = true }
  }, [satelliteId, enabled])

  useEffect(() => {
    let cancelled = false
    if (!satelliteId || !coordinates || !enabled) {
      setPasses([])
      return undefined
    }

    setPasses(null)
    fetchSatellitePasses(satelliteId, {
      latitudeDeg: coordinates.latitude,
      longitudeDeg: coordinates.longitude,
      altitudeM: coordinates.altitude,
      hours: 24,
      minElevationDeg: 0,
      limit: 3,
    })
      .then((data) => {
        if (!cancelled) setPasses(Array.isArray(data?.passes) ? data.passes : [])
      })
      .catch(() => {
        if (!cancelled) setPasses([])
      })

    return () => { cancelled = true }
  }, [satelliteId, coordinates?.latitude, coordinates?.longitude, coordinates?.altitude, enabled])

  return { orbitalEvents, passes }
}
