import { Html } from '@react-three/drei'
import { useI18n } from '../../lib/I18nContext'
import type { ClusterCentroid } from '../../lib/correlationLayout3d'
import { kindColor, type ScenePalette } from './sceneTheme'

interface ClusterMarkersProps {
  clusters: ClusterCentroid[]
  palette: ScenePalette
  minimumSize: number
}

export default function ClusterMarkers({ clusters, palette, minimumSize }: ClusterMarkersProps) {
  const { t } = useI18n()
  const visible = clusters.filter((cluster) => cluster.count >= minimumSize)

  return (
    <group>
      {visible.map((cluster) => {
        const tone = kindColor(cluster.kind, palette)
        return (
          <Html
            key={cluster.kind}
            position={[cluster.position.x, cluster.position.y - 3.4, cluster.position.z]}
            center
            distanceFactor={52}
            zIndexRange={[10, 0]}
            style={{ pointerEvents: 'none' }}
          >
            <span className="lyra-cluster-label" style={{ color: tone, borderColor: tone }}>
              {t(`correlationGraph.kind.${cluster.kind}`)}
              <span className="lyra-cluster-label__count">{cluster.count}</span>
            </span>
          </Html>
        )
      })}
    </group>
  )
}
