import { useMemo } from 'react'
import DottedMap from 'dotted-map/without-countries'
import worldMapData from '../lib/world-map-data.json'

export default function WorldMap({ satellites }) {
  const svg = useMemo(() => {
    const map = new DottedMap({ map: worldMapData })

    satellites.forEach((s) => {
      if (!s.position) return
      map.addPin({
        lat: s.position.latitude_deg,
        lng: s.position.longitude_deg,
        svgOptions: { color: '#f0d9a8', radius: 0.7 },
      })
    })

    return map.getSVG({
      radius: 0.24,
      color: '#332b56',
      shape: 'circle',
      backgroundColor: 'transparent',
    })
  }, [satellites])

  return (
    <div
      className="w-full [&_svg]:w-full [&_svg]:h-auto"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}
