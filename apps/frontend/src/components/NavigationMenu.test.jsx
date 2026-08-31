import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import NavigationMenu from './NavigationMenu.jsx'

describe('NavigationMenu', () => {
  const items = [
    { id: 'orbital', label: 'Orbital' },
    { id: 'assets', label: 'Assets' },
    { id: 'settings', label: 'Settings' },
  ]

  it('opens from the top-left control and navigates to a selected page', () => {
    const onChange = vi.fn()
    render(<NavigationMenu items={items} active="orbital" onChange={onChange} />)

    const trigger = screen.getByRole('button', { name: 'Open navigation' })
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Default dashboard')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('menuitem', { name: /Assets/i }))
    expect(onChange).toHaveBeenCalledWith('assets')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })

  it('marks the active page', () => {
    render(<NavigationMenu items={items} active="assets" onChange={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: 'Open navigation' }))
    expect(screen.getByRole('menuitem', { name: /Assets/i })).toHaveAttribute('aria-current', 'page')
  })
})
