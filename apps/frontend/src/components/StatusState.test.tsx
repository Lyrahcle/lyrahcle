import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import StatusState from './StatusState'

describe('StatusState', () => {
  it('uses a polite busy status for loading', () => {
    render(<StatusState kind="loading" title="Loading data" />)
    expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'true')
  })

  it('uses an alert for failures and exposes retry', () => {
    const retry = vi.fn()
    render(<StatusState kind="error" title="Data could not be loaded" message="backend unavailable" actionLabel="Retry" onAction={retry} />)
    expect(screen.getByRole('alert')).toHaveTextContent('backend unavailable')
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(retry).toHaveBeenCalledOnce()
  })

  it('does not announce a normal empty state as an alert', () => {
    render(<StatusState kind="empty" title="Nothing here yet" />)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.getByText('Nothing here yet').closest('section')).toHaveAttribute('data-state-kind', 'empty')
  })
})
