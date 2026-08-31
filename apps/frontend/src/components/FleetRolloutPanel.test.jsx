import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FleetRolloutPanel from './FleetRolloutPanel.jsx'
import { agentRolloutAction, fetchAgentRollout, fetchAgentRollouts } from '../lib/api.js'

vi.mock('../lib/api.js', () => ({
  fetchAgentRollouts: vi.fn(async () => []),
  fetchAgentRollout: vi.fn(),
  createAgentRollout: vi.fn(),
  agentRolloutAction: vi.fn(),
}))

const running = {
  id: 'rollout-1', name: 'Canary alpha', status: 'running', batch_size: 2,
  failure_threshold_percent: 20, soak_seconds: 300,
  targets: [
    { agent_id: 'station-a', ordinal: 0, status: 'applied', config_version: 4, issued_at: new Date(Date.now() - 600000).toISOString() },
    { agent_id: 'station-b', ordinal: 1, status: 'applied', config_version: 4, issued_at: new Date(Date.now() - 600000).toISOString() },
    { agent_id: 'station-c', ordinal: 2, status: 'issued', config_version: 4, issued_at: new Date(Date.now() - 10000).toISOString() },
    { agent_id: 'station-d', ordinal: 3, status: 'pending' },
  ],
}

describe('FleetRolloutPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fetchAgentRollouts.mockResolvedValue([])
  })

  it('exposes staged rollout controls in the UI', async () => {
    render(<FleetRolloutPanel agents={[{ agent_id: 'station-a', status: 'active' }]} canManage />)
    expect(await screen.findByText('Fleet staged rollouts')).toBeInTheDocument()
    expect(screen.getByText('station-a')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create staged rollout' })).toBeDisabled()
    expect(screen.getByLabelText('Failure threshold %')).toBeInTheDocument()
    expect(screen.getByLabelText('Soak target (s)')).toBeInTheDocument()
  })

  it('shows cohort progress and blocks promotion while the active batch is unacknowledged', async () => {
    fetchAgentRollouts.mockResolvedValue([{ id: 'rollout-1', name: 'Canary alpha', status: 'running' }])
    fetchAgentRollout.mockResolvedValue(running)
    render(<FleetRolloutPanel agents={[]} canManage />)
    fireEvent.click(await screen.findByRole('button', { name: 'Canary alpha' }))
    expect(await screen.findByText('2/4 · 50%')).toBeInTheDocument()
    expect(screen.getByText('#2')).toBeInTheDocument()
    expect(screen.getByLabelText('Soak timer')).toBeInTheDocument()
    expect(screen.getByText('cohort 2 · config v4')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Advance batch' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Advance batch' })).toHaveAttribute('title', 'Wait for the active batch to acknowledge')
  })

  it('requires confirmation before rollback', async () => {
    fetchAgentRollouts.mockResolvedValue([{ id: 'rollout-1', name: 'Canary alpha', status: 'running' }])
    fetchAgentRollout.mockResolvedValue({ ...running, targets: running.targets.map((target) => ({ ...target, status: 'applied' })) })
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<FleetRolloutPanel agents={[]} canManage />)
    fireEvent.click(await screen.findByRole('button', { name: 'Canary alpha' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Rollback' }))
    expect(confirmSpy).toHaveBeenCalled()
    expect(agentRolloutAction).not.toHaveBeenCalled()
    confirmSpy.mockRestore()
  })
})
