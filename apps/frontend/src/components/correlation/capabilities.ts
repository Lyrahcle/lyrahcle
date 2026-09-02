export function supportsWebGl(): boolean {
  if (typeof document === 'undefined' || typeof window === 'undefined') return false
  const contextConstructor = (window as { WebGLRenderingContext?: unknown }).WebGLRenderingContext
  if (typeof contextConstructor === 'undefined') return false
  try {
    const canvas = document.createElement('canvas')
    if (typeof canvas.getContext !== 'function') return false
    const context =
      canvas.getContext('webgl2') || canvas.getContext('webgl') || canvas.getContext('experimental-webgl')
    return Boolean(context)
  } catch {
    return false
  }
}

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
  try {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  } catch {
    return false
  }
}
