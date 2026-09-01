import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import CopyButton from './CopyButton.jsx'
import { I18nProvider } from '../lib/I18nContext'

function renderButton(props = {}) {
  return render(<I18nProvider><CopyButton value="25544" label="NORAD ID" {...props} /></I18nProvider>)
}

describe('CopyButton', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('copies the given value to the clipboard', async () => {
    renderButton()
    fireEvent.click(screen.getByRole('button', { name: /copy norad id/i }))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalledWith('25544'))
    await screen.findByRole('button', { name: /^copied$/i })
  })

  it('shows a confirmation state that reverts after a short delay', async () => {
    renderButton()
    fireEvent.click(screen.getByRole('button', { name: /copy norad id/i }))
    expect(await screen.findByRole('button', { name: /^copied$/i })).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(1600)
    })
    await act(async () => {})
    expect(await screen.findByRole('button', { name: /copy norad id/i })).toBeInTheDocument()
  })

  it('does not attempt to copy an empty value', () => {
    renderButton({ value: '' })
    fireEvent.click(screen.getByRole('button', { name: /copy norad id/i }))
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled()
  })

  it('falls back silently if the clipboard API rejects', async () => {
    navigator.clipboard.writeText.mockRejectedValueOnce(new Error('denied'))
    renderButton()
    fireEvent.click(screen.getByRole('button', { name: /copy norad id/i }))
    await waitFor(() => expect(navigator.clipboard.writeText).toHaveBeenCalled())
    await act(async () => {})
    expect(screen.getByRole('button', { name: /copy norad id/i })).toBeInTheDocument()
  })
})
