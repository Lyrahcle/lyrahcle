import { lazy, Suspense } from 'react'

const defaultFallback = (
  <div
    role="status"
    aria-live="polite"
    aria-label="Loading view"
    className="min-h-32 animate-pulse rounded-lg border border-slate-800/60 bg-slate-950/20"
  />
)

export function lazyComponent(loader, fallback = defaultFallback) {
  const LazyComponent = lazy(loader)

  function LazyBoundary(props) {
    return (
      <Suspense fallback={fallback}>
        <LazyComponent {...props} />
      </Suspense>
    )
  }

  LazyBoundary.displayName = 'LazyBoundary'
  return LazyBoundary
}
