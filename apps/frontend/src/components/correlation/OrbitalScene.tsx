import { useMemo } from 'react'
import { Canvas } from '@react-three/fiber'
import { Stars } from '@react-three/drei'
import { AdditiveBlending, DoubleSide } from 'three'
import type { CorrelationEdge } from '../../lib/correlationGraph'
import type { OrbitalLayout, Vec3 } from '../../lib/correlationLayout3d'
import ClusterMarkers from './ClusterMarkers'
import EvidenceLinks from './EvidenceLinks'
import FocusRig from './FocusRig'
import NodeMesh from './NodeMesh'
import RoutePath from './RoutePath'
import SmartLabels from './SmartLabels'
import { SCENE, readScenePalette } from './sceneTheme'

interface OrbitalSceneProps {
  layout: OrbitalLayout
  edges: CorrelationEdge[]
  selectedId: string | null
  hoveredId: string | null
  path: string[]
  routeKeys: Set<string>
  neighborIds: Set<string>
  isolate: boolean
  showClusters: boolean
  autoRotate: boolean
  reducedMotion: boolean
  focus: Vec3 | null
  focusToken: string
  onSelect: (id: string) => void
  onHover: (id: string | null) => void
  onBackground: () => void
  onUserInteract: () => void
  hopLabel: (index: number) => string
}

function OrbitGuides({ radii, color }: { radii: number[]; color: string }) {
  return (
    <group>
      {radii
        .filter((radius) => radius > 0)
        .map((radius, index) => (
          <mesh key={radius} rotation={[Math.PI / 2, 0, index * 0.22]}>
            <ringGeometry args={[radius - 0.035, radius + 0.035, 128]} />
            <meshBasicMaterial
              color={color}
              transparent
              opacity={index === 0 ? 0.2 : 0.1}
              side={DoubleSide}
              depthWrite={false}
              blending={AdditiveBlending}
            />
          </mesh>
        ))}
    </group>
  )
}

export default function OrbitalScene({
  layout,
  edges,
  selectedId,
  hoveredId,
  path,
  routeKeys,
  neighborIds,
  isolate,
  showClusters,
  autoRotate,
  reducedMotion,
  focus,
  focusToken,
  onSelect,
  onHover,
  onBackground,
  onUserInteract,
  hopLabel,
}: OrbitalSceneProps) {
  const palette = useMemo(() => readScenePalette(), [])
  const routeIds = useMemo(() => new Set(path), [path])

  return (
    <Canvas
      dpr={[1, 2]}
      camera={{ position: SCENE.cameraStart, fov: SCENE.cameraFov, near: 0.5, far: 900 }}
      gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}
      onPointerMissed={onBackground}
      style={{ touchAction: 'none' }}
    >
      <ambientLight intensity={0.72} />
      <directionalLight position={[24, 34, 18]} intensity={1.15} color={palette.text} />
      <pointLight position={[-30, -12, -22]} intensity={0.5} color={palette.gold} />

      {!reducedMotion && <Stars radius={220} depth={70} count={1400} factor={3.4} saturation={0} fade speed={0.4} />}

      <OrbitGuides radii={layout.ringRadii} color={palette.gold} />

      <EvidenceLinks
        edges={edges}
        byId={layout.byId}
        palette={palette}
        selectedId={selectedId}
        routeKeys={routeKeys}
        isolate={isolate}
        neighborIds={neighborIds}
      />

      <RoutePath
        path={path}
        byId={layout.byId}
        palette={palette}
        reducedMotion={reducedMotion}
        hopLabel={hopLabel}
      />

      {layout.placements.map((placement) => (
        <NodeMesh
          key={placement.node.id}
          placement={placement}
          palette={palette}
          selected={placement.node.id === selectedId}
          hovered={placement.node.id === hoveredId}
          onRoute={routeIds.has(placement.node.id)}
          dimmed={
            (isolate && !neighborIds.has(placement.node.id) && !routeIds.has(placement.node.id)) ||
            (routeIds.size > 1 && !routeIds.has(placement.node.id) && placement.node.id !== selectedId)
          }
          reducedMotion={reducedMotion}
          onSelect={onSelect}
          onHover={onHover}
        />
      ))}

      {showClusters && <ClusterMarkers clusters={layout.clusters} palette={palette} minimumSize={2} />}

      <SmartLabels
        placements={layout.placements}
        palette={palette}
        selectedId={selectedId}
        hoveredId={hoveredId}
        routeIds={routeIds}
        keyNodeLimit={isolate ? 10 : 5}
      />

      <FocusRig
        focus={focus}
        focusToken={focusToken}
        extent={layout.extent}
        autoRotate={autoRotate && !reducedMotion}
        onUserInteract={onUserInteract}
      />
    </Canvas>
  )
}
