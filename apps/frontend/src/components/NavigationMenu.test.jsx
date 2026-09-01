import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import NavigationMenu from './NavigationMenu.jsx'

function renderMenu(props) {
  return render(
    <I18nProvider>
      <NavigationMenu {...props} />
    </I18nProvider>,
  )
}

const items = [
  { id: 'orbital', label: 'Mission overview', section: 'overview' },
  { id: 'assets', label: 'Assets', section: 'mission' },
  { id: 'incidents', label: 'Incidents', section: 'security' },
  { id: 'settings', label: 'Settings', section: 'preferences' },
]

describe('NavigationMenu', () => {
  it('renders grouped desktop navigation and changes pages', () => {
    const onChange = vi.fn()
    renderMenu({ items, active: 'assets', onChange, collapsed: false, onToggleCollapse: vi.fn(), mobileOpen: false, onCloseMobile: vi.fn() })

    expect(screen.getByText('Overview')).toBeInTheDocument()
    expect(screen.getByText('Mission')).toBeInTheDocument()
    expect(screen.getByText('Security')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Assets' })).toHaveAttribute('aria-current', 'page')

    fireEvent.click(screen.getByRole('button', { name: 'Incidents' }))
    expect(onChange).toHaveBeenCalledWith('incidents')
  })

  it('supports the compact desktop sidebar', () => {
    const onToggleCollapse = vi.fn()
    renderMenu({ items, active: 'orbital', onChange: vi.fn(), collapsed: true, onToggleCollapse, mobileOpen: false, onCloseMobile: vi.fn() })

    expect(screen.getByRole('button', { name: 'Mission overview' })).toHaveAttribute('aria-current', 'page')
    fireEvent.click(screen.getByRole('button', { name: 'Expand sidebar' }))
    expect(onToggleCollapse).toHaveBeenCalledTimes(1)
  })

  it('opens an accessible mobile drawer and closes after navigation', () => {
    const onChange = vi.fn()
    const onCloseMobile = vi.fn()
    renderMenu({ items, active: 'orbital', onChange, collapsed: false, onToggleCollapse: vi.fn(), mobileOpen: true, onCloseMobile })

    const mobileAside = screen.getAllByRole('complementary')[1]
    fireEvent.click(within(mobileAside).getByRole('button', { name: 'Assets' }))
    expect(onChange).toHaveBeenCalledWith('assets')
    expect(onCloseMobile).toHaveBeenCalledTimes(1)
  })
})
