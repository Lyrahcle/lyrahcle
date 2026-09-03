import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import EvidenceBadge from './EvidenceBadge'
import { I18nProvider } from '../lib/I18nContext'

it('renders a translated evidence kind', () => {
  render(<I18nProvider><EvidenceBadge kind="context" /></I18nProvider>)
  expect(screen.getByText(/context/i)).toBeInTheDocument()
})
