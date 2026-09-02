import { useMemo, useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import { Html, Line } from '@react-three/drei'
import type { Mesh } from 'three'
import { CatmullRomCurve3, Vector3 } from 'three'
import { linkControlPoint, quadraticPoints, type LayoutPlacement } from '../../lib/correlationLayout3d'
import type { ScenePalette } from './sceneTheme'

interface RoutePathProps {
  path: string[]
  byId: Map<string, LayoutPlacement>
  palette: ScenePalette
  reducedMotion: boolean
  hopLabel: (index: number) => string
}

export default function RoutePath({ path, byId, palette, reducedMotion, hopLabel }: RoutePathProps) {
  const marker = useRef<Mesh>(null)

  const { points, curve } = useMemo(() => {
    const collected: [number, number, number][] = []
    for (let index = 0; index < path.length - 1; index += 1) {
      const source = byId.get(path[index])
      const target = byId.get(path[index + 1])
      if (!source || !target) continue
      const control = linkControlPoint(source.position, target.position)
      const segment = quadraticPoints(source.position, control, target.position, 26)
      for (const point of segment) collected.push([point.x, point.y, point.z])
    }
    if (collected.length < 2) return { points: [] as [number, number, number][], curve: null }
    return {
      points: collected,
      curve: new CatmullRomCurve3(collected.map(([x, y, z]) => new Vector3(x, y, z))),
    }
  }, [path, byId])

  useFrame((state) => {
    if (!marker.current || !curve || reducedMotion) return
    const t = (state.clock.elapsedTime * 0.16) % 1
    const position = curve.getPointAt(t)
    marker.current.position.copy(position)
  })

  const hops = useMemo(
    () => path.map((id) => byId.get(id)).filter((placement): placement is LayoutPlacement => Boolean(placement)),
    [path, byId],
  )

  if (points.length < 2) return null

  return (
    <group>
      <Line points={points} color={palette.goldBright} lineWidth={2.6} transparent opacity={0.96} />
      <Line points={points} color={palette.gold} lineWidth={7} transparent opacity={0.16} />
      {hops.map((placement, index) => (
        <Html
          key={placement.node.id}
          position={[placement.position.x, placement.position.y - 2.2, placement.position.z]}
          center
          distanceFactor={34}
          zIndexRange={[30, 0]}
          style={{ pointerEvents: 'none' }}
        >
          <span className="lyra-route-hop" style={{ color: palette.goldBright, borderColor: palette.gold }}>
            {hopLabel(index + 1)}
          </span>
        </Html>
      ))}
      {!reducedMotion && (
        <mesh ref={marker}>
          <sphereGeometry args={[0.32, 16, 16]} />
          <meshBasicMaterial color={palette.goldBright} />
        </mesh>
      )}
    </group>
  )
}
