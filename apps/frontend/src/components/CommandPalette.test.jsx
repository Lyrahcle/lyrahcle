import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import CommandPalette from './CommandPalette.jsx'

vi.mock('../lib/api.js', () => ({
  globalSearch: vi.fn(),
}))

import { globalSearch } from '../lib/api.js'

const navigationItems = [
  { id: 'orbital', label: 'Mission overview', section: 'overview' },
  { id: 'stations', label: 'Ground stations', section: 'mission' },
  { id: 'incidents', label: 'Incidents', section: 'security' },
]

function renderPalette(onNavigate = vi.fn()) {
  render(
    <I18nProvider>
      <CommandPalette navigationItems={navigationItems} onNavigate={onNavigate} />
    </I18nProvider>,
  )
  return onNavigate
}

describe('CommandPalette', () => {
  beforeEach(() => {
    globalSearch.mockReset()
    globalSearch.mockResolvedValue([])
  })

  it('opens with Ctrl+K and jumps to an allowed navigation destination', () => {
    const onNavigate = renderPalette()
    fireEvent.keyDown(document, { key: 'k', ctrlKey: true })
    expect(screen.getByRole('dialog', { name: 'Command palette' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('option', { name: /Ground stations/i }))
    expect(onNavigate).toHaveBeenCalledWith('stations', null)
    expect(screen.queryByRole('dialog', { name: 'Command palette' })).not.toBeInTheDocument()
  })

  it('supports arrow and Enter keyboard navigation', () => {
    const onNavigate = renderPalette()
    fireEvent.click(screen.getByRole('button', { name: 'Open command palette' }))
    const input = screen.getByRole('searchbox', { name: 'Search or jump to a destination' })
    fireEvent.keyDown(input, { key: 'ArrowDown' })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onNavigate).toHaveBeenCalledWith('stations', null)
  })

  it('searches globally and returns a satellite result for direct profile navigation', async () => {
    globalSearch.mockResolvedValue([
      { type: 'satellites', id: 'sat-1', label: 'NOAA-19', subtitle: 'NORAD 33591', tab: 'orbital' },
      { type: 'assets', id: 'asset-1', label: 'Hidden asset', subtitle: 'receiver', tab: 'assets' },
    ])
    const onNavigate = renderPalette()
    fireEvent.click(screen.getByRole('button', { name: 'Open command palette' }))
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'NOAA' } })
    await waitFor(() => expect(globalSearch).toHaveBeenCalledWith('NOAA', expect.any(AbortSignal)))
    const satellite = await screen.findByRole('option', { name: /NOAA-19/i })
    expect(screen.queryByText('Hidden asset')).not.toBeInTheDocument()
    fireEvent.click(satellite)
    expect(onNavigate).toHaveBeenCalledWith('orbital', expect.objectContaining({ type: 'satellites', id: 'sat-1' }))
  })
})
