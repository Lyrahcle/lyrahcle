import { lazy, Suspense, type ComponentType, type ReactNode } from 'react'

const defaultFallback: ReactNode = (
  <div
    role="status"
    aria-live="polite"
    aria-label="Loading view"
    className="min-h-32 animate-pulse rounded-lg border border-slate-800/60 bg-slate-950/20"
  />
)

export function lazyComponent<P extends object>(
  loader: () => Promise<{ default: ComponentType<P> }>,
  fallback: ReactNode = defaultFallback,
): ComponentType<P> {
  const LazyComponent = lazy(loader)

  function LazyBoundary(props: P) {
    const Component = LazyComponent as unknown as ComponentType<P>
    return (
      <Suspense fallback={fallback}>
        <Component {...props} />
      </Suspense>
    )
  }

  LazyBoundary.displayName = 'LazyBoundary'
  return LazyBoundary
}
