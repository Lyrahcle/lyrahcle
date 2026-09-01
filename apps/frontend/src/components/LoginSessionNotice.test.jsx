import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import Login from './Login.jsx'

describe('Login session notice', () => {
  it('explains that an expired or revoked session requires sign-in again', () => {
    window.history.replaceState({}, '', '/')
    render(<I18nProvider><Login notice="session_expired" onSuccess={() => {}} /></I18nProvider>)
    expect(screen.getByRole('status')).toHaveTextContent(/session expired or was revoked/i)
  })
})
