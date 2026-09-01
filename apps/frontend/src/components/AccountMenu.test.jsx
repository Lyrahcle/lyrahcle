import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import AccountMenu from './AccountMenu.jsx'

function renderMenu(onLogout = vi.fn()) {
  return render(
    <I18nProvider>
      <AccountMenu onLogout={onLogout} />
    </I18nProvider>,
  )
}

describe('AccountMenu', () => {
  it('exposes the dropdown as an ARIA menu with menuitem entries', () => {
    renderMenu()
    const trigger = screen.getByRole('button', { name: /account|admin|operator|viewer/i })
    expect(trigger).toHaveAttribute('aria-haspopup', 'menu')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')

    const menu = screen.getByRole('menu')
    expect(menu).toBeInTheDocument()
    const items = screen.getAllByRole('menuitem')
    expect(items.length).toBeGreaterThanOrEqual(4)
  })

  it('closes the menu and calls onLogout when Sign out is chosen', () => {
    const onLogout = vi.fn()
    renderMenu(onLogout)
    fireEvent.click(screen.getByRole('button', { name: /account|admin|operator|viewer/i }))
    fireEvent.click(screen.getByRole('menuitem', { name: /sign out/i }))
    expect(onLogout).toHaveBeenCalled()
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('closes on Escape', () => {
    renderMenu()
    fireEvent.click(screen.getByRole('button', { name: /account|admin|operator|viewer/i }))
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })
})
