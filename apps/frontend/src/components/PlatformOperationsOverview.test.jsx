import { render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PlatformOperationsOverview from './PlatformOperationsOverview.jsx'

vi.mock('../lib/api.js', () => ({
  fetchPlatformReadiness: vi.fn(),
  fetchPlatformSecretStatus: vi.fn(),
  fetchAgentConfigSigningKeys: vi.fn(),
}))
vi.mock('../lib/I18nContext.jsx', () => ({ useI18n: () => ({ t: (key) => key.split('.').pop() }) }))

import { fetchAgentConfigSigningKeys, fetchPlatformReadiness, fetchPlatformSecretStatus } from '../lib/api.js'

describe('PlatformOperationsOverview', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows a healthy control-plane posture', async () => {
    fetchPlatformReadiness.mockResolvedValue({ status: 'ready', ok: true })
    fetchPlatformSecretStatus.mockResolvedValue({ provider: 'remote', healthy: true, external: true })
    fetchAgentConfigSigningKeys.mockResolvedValue({ managed: true, health: { rotation_due: false, warnings: [] } })
    render(<PlatformOperationsOverview organizations={[{ id: '1' }, { id: '2' }]} platformUsers={[{ platform_admin: true }, { platform_admin: false }]} />)
    expect(screen.getByRole('heading', { name: 'title' })).toBeInTheDocument()
    await waitFor(() => expect(screen.getAllByText('healthy').length).toBeGreaterThanOrEqual(3))
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('degrades when readiness or custody needs attention', async () => {
    fetchPlatformReadiness.mockResolvedValue({ status: 'not_ready', ok: false })
    fetchPlatformSecretStatus.mockResolvedValue({ healthy: false })
    fetchAgentConfigSigningKeys.mockResolvedValue({ managed: true, health: { rotation_due: true, warnings: [{ code: 'rotation_due' }] } })
    render(<PlatformOperationsOverview organizations={[]} platformUsers={[]} />)
    await waitFor(() => expect(screen.getAllByText('degraded').length).toBeGreaterThanOrEqual(1))
    expect(screen.getByText('notReady')).toBeInTheDocument()
    expect(screen.getByText('attention')).toBeInTheDocument()
  })
})
