import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import SignalBar from './SignalBar.jsx'

describe('SignalBar', () => {
  it('renders the dBm value formatted to one decimal', () => {
    render(<SignalBar dbm={-42.34} />)
    expect(screen.getByText('-42.3 dBm')).toBeInTheDocument()
  })

  it('uses the good tone for a strong signal', () => {
    const { container } = render(<SignalBar dbm={-30} />)
    const bar = container.querySelector('.bg-lyra-good')
    expect(bar).not.toBeNull()
  })

  it('uses the warn tone for a mid-strength signal', () => {
    const { container } = render(<SignalBar dbm={-65} />)
    const bar = container.querySelector('.bg-lyra-warn')
    expect(bar).not.toBeNull()
  })

  it('uses the bad tone for a weak signal', () => {
    const { container } = render(<SignalBar dbm={-100} />)
    const bar = container.querySelector('.bg-lyra-bad')
    expect(bar).not.toBeNull()
  })

  it('clamps the bar width at 100% for signals stronger than the max', () => {
    const { container } = render(<SignalBar dbm={10} />)
    const bar = container.querySelector('.bg-lyra-good')
    expect(bar.style.width).toBe('100%')
  })

  it('clamps the bar width at 0% for signals weaker than the min', () => {
    const { container } = render(<SignalBar dbm={-200} />)
    const bar = container.querySelector('.bg-lyra-bad')
    expect(bar.style.width).toBe('0%')
  })
})
