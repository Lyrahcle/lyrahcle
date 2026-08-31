import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FleetRolloutPanel from './FleetRolloutPanel.jsx'

vi.mock('../lib/api.js', () => ({
  fetchAgentRollouts: vi.fn(async () => []),
  fetchAgentRollout: vi.fn(),
  createAgentRollout: vi.fn(),
  agentRolloutAction: vi.fn(),
}))

describe('FleetRolloutPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('exposes staged rollout controls in the UI', async () => {
    render(<FleetRolloutPanel agents={[{ agent_id: 'station-a', status: 'active' }]} canManage />)
    expect(await screen.findByText('Fleet staged rollouts')).toBeInTheDocument()
    expect(screen.getByText('station-a')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create staged rollout' })).toBeDisabled()
    expect(screen.getByLabelText('Failure threshold %')).toBeInTheDocument()
    expect(screen.getByLabelText('Soak target (s)')).toBeInTheDocument()
  })
})
