import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import KindBadge from './KindBadge.jsx'

describe('KindBadge', () => {
  it('renders the kind text', () => {
    render(<KindBadge kind="domain" />)
    expect(screen.getByText('domain')).toBeInTheDocument()
  })

  it('applies a distinct color per known kind', () => {
    const kinds = ['domain', 'ip', 'satellite', 'ground_station', 'firmware']
    const classes = kinds.map((kind) => {
      const { container, unmount } = render(<KindBadge kind={kind} />)
      const cls = container.querySelector('span').className
      unmount()
      return cls
    })
    expect(new Set(classes).size).toBe(kinds.length)
  })

  it('falls back to the firmware style for an unknown kind', () => {
    const { container } = render(<KindBadge kind="something-unexpected" />)
    expect(container.querySelector('span').className).toContain('text-lyra-muted')
  })
})
