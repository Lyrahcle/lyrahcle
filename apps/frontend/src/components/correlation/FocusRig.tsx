import { useEffect, useRef } from 'react'
import { useFrame, useThree } from '@react-three/fiber'
import { OrbitControls } from '@react-three/drei'
import { Vector3 } from 'three'
import type { Vec3 } from '../../lib/correlationLayout3d'
import { SCENE } from './sceneTheme'

interface FocusRigProps {
  focus: Vec3 | null
  focusToken: string
  extent: number
  autoRotate: boolean
  onUserInteract: () => void
}

export default function FocusRig({ focus, focusToken, extent, autoRotate, onUserInteract }: FocusRigProps) {
  const controls = useRef<any>(null)
  const { camera } = useThree()
  const desiredTarget = useRef(new Vector3(0, 0, 0))
  const desiredPosition = useRef(new Vector3(...SCENE.cameraStart))
  const settling = useRef(false)

  useEffect(() => {
    if (!focus) return
    const target = new Vector3(focus.x, focus.y, focus.z)
    desiredTarget.current.copy(target)
    const direction = camera.position.clone().sub(target)
    if (direction.lengthSq() < 1) direction.set(0, 0.4, 1)
    direction.normalize()
    const distance = Math.max(SCENE.focusDistance, Math.min(SCENE.maxDistance, extent * 1.55))
    desiredPosition.current.copy(target).add(direction.multiplyScalar(distance))
    settling.current = true
  }, [focusToken, focus, camera, extent])

  useFrame((_, delta) => {
    const instance = controls.current
    if (!instance) return
    if (settling.current) {
      const step = Math.min(1, delta * 3.2)
      instance.target.lerp(desiredTarget.current, step)
      camera.position.lerp(desiredPosition.current, step)
      if (
        instance.target.distanceTo(desiredTarget.current) < 0.05 &&
        camera.position.distanceTo(desiredPosition.current) < 0.4
      ) {
        settling.current = false
      }
    }
    instance.update()
  })

  return (
    <OrbitControls
      ref={controls}
      makeDefault
      enablePan={false}
      enableDamping
      dampingFactor={0.075}
      rotateSpeed={0.62}
      zoomSpeed={0.7}
      autoRotate={autoRotate}
      autoRotateSpeed={0.28}
      minDistance={SCENE.minDistance}
      maxDistance={SCENE.maxDistance}
      onStart={() => {
        settling.current = false
        onUserInteract()
      }}
    />
  )
}
