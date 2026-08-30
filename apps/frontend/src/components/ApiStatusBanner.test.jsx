import { act, fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import ApiStatusBanner from './ApiStatusBanner.jsx'

function renderBanner(issue, props = {}) {
  return render(
    <I18nProvider>
      <ApiStatusBanner issue={issue} onRetry={props.onRetry} onDismiss={props.onDismiss ?? vi.fn()} />
    </I18nProvider>,
  )
}

describe('ApiStatusBanner', () => {
  it('renders an actionable forbidden state without a retry button', () => {
    renderBanner({ kind: 'forbidden', status: 403, retryable: false, requestId: 'req-403' })
    expect(screen.getByText(/permission denied/i)).toBeInTheDocument()
    expect(screen.getByText(/req-403/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^retry$/i })).not.toBeInTheDocument()
  })

  it('offers retry for backend/network failures', () => {
    const onRetry = vi.fn()
    renderBanner({ kind: 'network', status: 0, retryable: true }, { onRetry })
    fireEvent.click(screen.getByRole('button', { name: /^retry$/i }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('respects Retry-After before enabling a rate-limit retry', () => {
    vi.useFakeTimers()
    const onRetry = vi.fn()
    renderBanner({ kind: 'rate_limited', status: 429, retryable: true, retryAfterSeconds: 2 }, { onRetry })
    const button = screen.getByRole('button', { name: /retry in 2s/i })
    expect(button).toBeDisabled()
    act(() => { vi.advanceTimersByTime(2100) })
    expect(screen.getByRole('button', { name: /^retry$/i })).toBeEnabled()
    vi.useRealTimers()
  })
})
