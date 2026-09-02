import type { CorrelationNodeKind, CorrelationSeverity } from '../../lib/correlationGraph'

export interface ScenePalette {
  gold: string
  goldBright: string
  text: string
  muted: string
  good: string
  warn: string
  bad: string
  border: string
  panel: string
}

const FALLBACK_PALETTE: ScenePalette = {
  gold: '#c9a56e',
  goldBright: '#f0d9a8',
  text: '#ecebeb',
  muted: '#8c8c90',
  good: '#7dd3a0',
  warn: '#e8b563',
  bad: '#e0748c',
  border: '#26262a',
  panel: '#141416',
}

function toHex(triplet: string, fallback: string): string {
  const parts = triplet.trim().split(/[\s,]+/).map(Number)
  if (parts.length < 3 || parts.some((value) => !Number.isFinite(value))) return fallback
  return `#${parts
    .slice(0, 3)
    .map((value) => Math.max(0, Math.min(255, Math.round(value))).toString(16).padStart(2, '0'))
    .join('')}`
}

export function readScenePalette(): ScenePalette {
  if (typeof window === 'undefined' || typeof getComputedStyle !== 'function') return FALLBACK_PALETTE
  const styles = getComputedStyle(document.documentElement)
  const read = (name: string, fallback: string) => toHex(styles.getPropertyValue(name) || '', fallback)
  return {
    gold: read('--lyra-gold', FALLBACK_PALETTE.gold),
    goldBright: read('--lyra-gold-bright', FALLBACK_PALETTE.goldBright),
    text: read('--lyra-text', FALLBACK_PALETTE.text),
    muted: read('--lyra-muted', FALLBACK_PALETTE.muted),
    good: read('--lyra-good', FALLBACK_PALETTE.good),
    warn: read('--lyra-warn', FALLBACK_PALETTE.warn),
    bad: read('--lyra-bad', FALLBACK_PALETTE.bad),
    border: read('--lyra-border', FALLBACK_PALETTE.border),
    panel: read('--lyra-panel', FALLBACK_PALETTE.panel),
  }
}

export function kindColor(kind: CorrelationNodeKind, palette: ScenePalette): string {
  if (kind === 'incident') return palette.bad
  if (kind === 'rf') return palette.warn
  if (kind === 'station' || kind === 'agent') return palette.good
  if (kind === 'cve') return palette.goldBright
  if (kind === 'asset' || kind === 'program') return palette.muted
  return palette.gold
}

export function severityColor(severity: CorrelationSeverity, palette: ScenePalette): string {
  if (severity === 'critical' || severity === 'high') return palette.bad
  if (severity === 'medium') return palette.warn
  if (severity === 'low') return palette.good
  return palette.border
}

export function kindScale(kind: CorrelationNodeKind): number {
  if (kind === 'satellite') return 1.16
  if (kind === 'incident' || kind === 'cve') return 1.02
  if (kind === 'station') return 1.06
  return 0.92
}

export const SCENE = {
  cameraFov: 42,
  cameraStart: [0, 21, 62] as [number, number, number],
  minDistance: 16,
  maxDistance: 148,
  focusDistance: 34,
  labelDistance: 78,
}
