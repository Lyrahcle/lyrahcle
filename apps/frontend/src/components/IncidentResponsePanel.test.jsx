import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import IncidentResponsePanel from './IncidentResponsePanel.tsx'

describe('IncidentResponsePanel', () => {
  it('surfaces the critical containment playbooks without exposing secret values', () => {
    render(<IncidentResponsePanel />)
    expect(screen.getByText('First 15 minutes')).toBeInTheDocument()
    expect(screen.getByText('Signing key')).toBeInTheDocument()
    expect(screen.getByText('Cross-tenant exposure')).toBeInTheDocument()
    expect(screen.getByText(/Never disable RLS/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /full incident response runbook/i })).toHaveAttribute('rel', 'noopener noreferrer')
    expect(document.body.textContent).not.toMatch(/password_hash|totp_secret|bearer token/i)
  })
})
