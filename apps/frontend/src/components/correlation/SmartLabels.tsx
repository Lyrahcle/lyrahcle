import { Html } from '@react-three/drei'
import type { LayoutPlacement } from '../../lib/correlationLayout3d'
import { kindColor, type ScenePalette } from './sceneTheme'

interface SmartLabelsProps {
  placements: LayoutPlacement[]
  palette: ScenePalette
  selectedId: string | null
  hoveredId: string | null
  routeIds: Set<string>
  keyNodeLimit: number
}

function truncate(value: string, length: number): string {
  return value.length > length ? `${value.slice(0, length - 1)}…` : value
}

export default function SmartLabels({
  placements,
  palette,
  selectedId,
  hoveredId,
  routeIds,
  keyNodeLimit,
}: SmartLabelsProps) {
  const hubs = new Set(
    [...placements]
      .filter((placement) => placement.degree > 0 && placement.ring <= 1)
      .sort((a, b) => b.degree - a.degree)
      .slice(0, keyNodeLimit)
      .map((placement) => placement.node.id),
  )

  const visible = placements.filter((placement) => {
    const id = placement.node.id
    return id === selectedId || id === hoveredId || routeIds.has(id) || hubs.has(id)
  })

  return (
    <group>
      {visible.map((placement) => {
        const id = placement.node.id
        const emphasised = id === selectedId || id === hoveredId
        const tone = id === selectedId ? palette.goldBright : kindColor(placement.node.kind, palette)
        return (
          <Html
            key={id}
            position={[placement.position.x, placement.position.y + 2.1, placement.position.z]}
            center
            distanceFactor={38}
            zIndexRange={[20, 0]}
            style={{ pointerEvents: 'none' }}
          >
            <span
              className="lyra-scene-label"
              style={{
                color: tone,
                opacity: emphasised ? 1 : 0.72,
                fontWeight: emphasised ? 600 : 450,
                borderColor: emphasised ? tone : 'transparent',
              }}
            >
              {truncate(placement.node.label, emphasised ? 30 : 18)}
            </span>
          </Html>
        )
      })}
    </group>
  )
}
