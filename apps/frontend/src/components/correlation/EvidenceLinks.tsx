import { useEffect, useMemo } from 'react'
import { AdditiveBlending, BufferAttribute, BufferGeometry } from 'three'
import type { CorrelationEdge } from '../../lib/correlationGraph'
import {
  edgeKey,
  linkControlPoint,
  quadraticPoints,
  type LayoutPlacement,
} from '../../lib/correlationLayout3d'
import type { ScenePalette } from './sceneTheme'

interface EvidenceLinksProps {
  edges: CorrelationEdge[]
  byId: Map<string, LayoutPlacement>
  palette: ScenePalette
  selectedId: string | null
  routeKeys: Set<string>
  isolate: boolean
  neighborIds: Set<string>
}

function buildGeometry(segments: number[][]): BufferGeometry | null {
  if (!segments.length) return null
  const vertices = new Float32Array(segments.flat())
  const geometry = new BufferGeometry()
  geometry.setAttribute('position', new BufferAttribute(vertices, 3))
  return geometry
}

export default function EvidenceLinks({
  edges,
  byId,
  palette,
  selectedId,
  routeKeys,
  isolate,
  neighborIds,
}: EvidenceLinksProps) {
  const { ambient, focused } = useMemo(() => {
    const ambientSegments: number[][] = []
    const focusedSegments: number[][] = []
    for (const edge of edges) {
      const source = byId.get(edge.source)
      const target = byId.get(edge.target)
      if (!source || !target) continue
      if (routeKeys.has(edgeKey(edge))) continue
      const touchesSelection = selectedId ? edge.source === selectedId || edge.target === selectedId : false
      if (isolate && !touchesSelection && !(neighborIds.has(edge.source) && neighborIds.has(edge.target))) continue
      const control = linkControlPoint(source.position, target.position)
      const points = quadraticPoints(source.position, control, target.position, touchesSelection ? 22 : 14)
      const bucket = touchesSelection ? focusedSegments : ambientSegments
      for (let index = 0; index < points.length - 1; index += 1) {
        const a = points[index]
        const b = points[index + 1]
        bucket.push([a.x, a.y, a.z, b.x, b.y, b.z])
      }
    }
    return { ambient: buildGeometry(ambientSegments), focused: buildGeometry(focusedSegments) }
  }, [edges, byId, selectedId, routeKeys, isolate, neighborIds])

  useEffect(
    () => () => {
      ambient?.dispose()
      focused?.dispose()
    },
    [ambient, focused],
  )

  return (
    <group>
      {ambient && (
        <lineSegments geometry={ambient}>
          <lineBasicMaterial
            color={palette.muted}
            transparent
            opacity={isolate ? 0.2 : 0.24}
            depthWrite={false}
            blending={AdditiveBlending}
          />
        </lineSegments>
      )}
      {focused && (
        <lineSegments geometry={focused}>
          <lineBasicMaterial
            color={palette.gold}
            transparent
            opacity={0.68}
            depthWrite={false}
            blending={AdditiveBlending}
          />
        </lineSegments>
      )}
    </group>
  )
}
