import { useMemo, useRef } from 'react'
import { useFrame } from '@react-three/fiber'
import type { Group, Mesh } from 'three'
import { DoubleSide } from 'three'
import type { LayoutPlacement } from '../../lib/correlationLayout3d'
import { kindColor, kindScale, type ScenePalette } from './sceneTheme'

interface NodeMeshProps {
  placement: LayoutPlacement
  palette: ScenePalette
  selected: boolean
  hovered: boolean
  onRoute: boolean
  dimmed: boolean
  reducedMotion: boolean
  onSelect: (id: string) => void
  onHover: (id: string | null) => void
}

function KindGeometry({ placement, color }: { placement: LayoutPlacement; color: string }) {
  const kind = placement.node.kind
  const surface = (
    <meshStandardMaterial
      color={color}
      emissive={color}
      emissiveIntensity={0.42}
      roughness={0.34}
      metalness={0.24}
    />
  )

  if (kind === 'satellite') {
    return (
      <group>
        <mesh>
          <sphereGeometry args={[0.66, 28, 28]} />
          {surface}
        </mesh>
        <mesh rotation={[Math.PI / 2.1, 0, 0.32]}>
          <torusGeometry args={[1.34, 0.045, 10, 72]} />
          <meshBasicMaterial color={color} transparent opacity={0.72} />
        </mesh>
        <mesh rotation={[Math.PI / 2.1, 0.9, 0.32]}>
          <torusGeometry args={[1.06, 0.03, 10, 64]} />
          <meshBasicMaterial color={color} transparent opacity={0.38} />
        </mesh>
      </group>
    )
  }

  if (kind === 'incident') {
    return (
      <mesh rotation={[0, Math.PI / 5, 0]}>
        <octahedronGeometry args={[0.98, 0]} />
        {surface}
      </mesh>
    )
  }

  if (kind === 'rf') {
    return (
      <group>
        <mesh>
          <octahedronGeometry args={[0.52, 0]} />
          {surface}
        </mesh>
        <mesh rotation={[Math.PI / 2, 0, 0]}>
          <ringGeometry args={[0.88, 0.94, 44]} />
          <meshBasicMaterial color={color} transparent opacity={0.42} side={DoubleSide} />
        </mesh>
      </group>
    )
  }

  if (kind === 'cve') {
    return (
      <group rotation={[Math.PI / 2.6, 0, 0]}>
        <mesh>
          <cylinderGeometry args={[0.94, 0.94, 0.11, 6]} />
          {surface}
        </mesh>
        <mesh>
          <cylinderGeometry args={[1.06, 1.06, 0.03, 6]} />
          <meshBasicMaterial color={color} transparent opacity={0.4} />
        </mesh>
      </group>
    )
  }

  if (kind === 'station' || kind === 'agent') {
    return (
      <group>
        <mesh position={[0, -0.22, 0]}>
          <cylinderGeometry args={[0.16, 0.24, 1.32, 12]} />
          {surface}
        </mesh>
        <mesh position={[0, 0.62, 0]} rotation={[-0.42, 0, 0]}>
          <sphereGeometry args={[0.52, 24, 16, 0, Math.PI * 2, 0, Math.PI / 2.1]} />
          <meshStandardMaterial
            color={color}
            emissive={color}
            emissiveIntensity={0.3}
            roughness={0.4}
            metalness={0.2}
            side={DoubleSide}
          />
        </mesh>
      </group>
    )
  }

  if (kind === 'program') {
    return (
      <mesh rotation={[0.32, 0.5, 0]}>
        <boxGeometry args={[1.24, 1.24, 1.24]} />
        {surface}
      </mesh>
    )
  }

  return (
    <mesh rotation={[0, 0.5, 0]}>
      <boxGeometry args={[1.06, 0.62, 1.06]} />
      {surface}
    </mesh>
  )
}

export default function NodeMesh({
  placement,
  palette,
  selected,
  hovered,
  onRoute,
  dimmed,
  reducedMotion,
  onSelect,
  onHover,
}: NodeMeshProps) {
  const group = useRef<Group>(null)
  const halo = useRef<Mesh>(null)
  const color = useMemo(() => kindColor(placement.node.kind, palette), [placement.node.kind, palette])
  const baseScale = kindScale(placement.node.kind)
  const target = baseScale * (selected ? 1.42 : hovered ? 1.18 : onRoute ? 1.12 : 1)

  useFrame((state, delta) => {
    const node = group.current
    if (!node) return
    const step = Math.min(1, delta * 7)
    node.scale.x += (target - node.scale.x) * step
    node.scale.y += (target - node.scale.y) * step
    node.scale.z += (target - node.scale.z) * step
    if (!reducedMotion) node.rotation.y += delta * (selected ? 0.24 : 0.06)
    if (halo.current) {
      halo.current.lookAt(state.camera.position)
      const pulse = reducedMotion ? 1 : 1 + Math.sin(state.clock.elapsedTime * 2.1) * 0.05
      halo.current.scale.setScalar(pulse)
    }
  })

  const faded = dimmed && !selected && !onRoute
  const opacity = faded ? 0.2 : 1

  return (
    <group
      ref={group}
      position={[placement.position.x, placement.position.y, placement.position.z]}
      onClick={(event) => {
        event.stopPropagation()
        onSelect(placement.node.id)
      }}
      onPointerOver={(event) => {
        event.stopPropagation()
        if (typeof document !== 'undefined') document.body.style.cursor = 'pointer'
        onHover(placement.node.id)
      }}
      onPointerOut={() => {
        if (typeof document !== 'undefined') document.body.style.cursor = ''
        onHover(null)
      }}
    >
      {faded ? (
        <mesh>
          <sphereGeometry args={[0.34, 14, 14]} />
          <meshBasicMaterial color={palette.muted} transparent opacity={opacity} depthWrite={false} />
        </mesh>
      ) : (
        <KindGeometry placement={placement} color={color} />
      )}
      {(selected || onRoute) && (
        <mesh ref={halo}>
          <ringGeometry args={[1.62, 1.74, 64]} />
          <meshBasicMaterial color={selected ? palette.goldBright : color} transparent opacity={selected ? 0.78 : 0.4} />
        </mesh>
      )}
      <mesh>
        <sphereGeometry args={[1.7, 8, 8]} />
        <meshBasicMaterial transparent opacity={0} depthWrite={false} colorWrite={false} />
      </mesh>
    </group>
  )
}
