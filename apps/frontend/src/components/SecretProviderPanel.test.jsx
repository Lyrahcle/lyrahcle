import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import SecretProviderPanel from './SecretProviderPanel.tsx'
import { fetchPlatformSecretStatus } from '../lib/api'

vi.mock('../lib/api', () => ({ fetchPlatformSecretStatus: vi.fn() }))

describe('SecretProviderPanel', () => {
  beforeEach(() => vi.clearAllMocks())
  it('shows provider metadata without secret values', async () => {
    fetchPlatformSecretStatus.mockResolvedValue({ provider:'remote', external:true, location:'https://secrets.internal', healthy:true, managed_keys:['JWT_SECRET'] })
    render(
      <I18nProvider>
        <SecretProviderPanel />
      </I18nProvider>,
    )
    expect(await screen.findByText('remote')).toBeInTheDocument()
    expect(screen.getByText('External custody')).toBeInTheDocument()
    expect(screen.getByText('JWT_SECRET')).toBeInTheDocument()
    expect(document.body.textContent).not.toContain('managed-secret')
  })
})
