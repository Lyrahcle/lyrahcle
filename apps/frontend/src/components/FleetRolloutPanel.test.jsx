import { fireEvent, render, screen, within, act } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import FleetRolloutPanel from './FleetRolloutPanel.jsx'
import { I18nProvider } from '../lib/I18nContext.jsx'
import { agentRolloutAction, createAgentRollout, fetchAgentRollout, fetchAgentRollouts } from '../lib/api.js'

const flushReactUpdates = async () => {
  await act(async () => {
    await Promise.resolve()
    await Promise.resolve()
  })
}

vi.mock('../lib/api.js', () => ({
  fetchAgentRollouts: vi.fn(async () => []),
  fetchAgentRollout: vi.fn(),
  createAgentRollout: vi.fn(),
  agentRolloutAction: vi.fn(),
}))

const running = {
  id: 'rollout-1', name: 'Canary alpha', status: 'running', batch_size: 2,
  failure_threshold_percent: 20, soak_seconds: 300, ack_timeout_seconds: 900,
  targets: [
    { agent_id: 'station-a', ordinal: 0, status: 'applied', config_version: 4, issued_at: new Date(Date.now() - 600000).toISOString() },
    { agent_id: 'station-b', ordinal: 1, status: 'applied', config_version: 4, issued_at: new Date(Date.now() - 600000).toISOString() },
    { agent_id: 'station-c', ordinal: 2, status: 'issued', config_version: 4, issued_at: new Date(Date.now() - 10000).toISOString() },
    { agent_id: 'station-d', ordinal: 3, status: 'pending' },
  ],
}

const compatibleAgent = {
  agent_id: 'station-good',
  status: 'active',
  device: {
    compatibility: 'compatible',
    compatibility_reason: 'software capability report satisfies the v1 baseline',
    capabilities: { frequency_min_hz: 24_000_000, frequency_max_hz: 1_766_000_000, integrity_status: 'healthy' },
  },
}

const changedAgent = {
  agent_id: 'station-changed',
  status: 'active',
  device: {
    compatibility: 'compatible',
    capabilities: { frequency_min_hz: 24_000_000, frequency_max_hz: 1_766_000_000, integrity_status: 'changed' },
  },
}

describe('FleetRolloutPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fetchAgentRollouts.mockResolvedValue([])
  })

  it('exposes staged rollout controls in the UI', async () => {
    render(<I18nProvider><FleetRolloutPanel agents={[{ agent_id: 'station-a', status: 'active' }]} canManage /></I18nProvider>)
    expect(await screen.findByText('Fleet staged rollouts')).toBeInTheDocument()
    expect(screen.getByText('station-a')).toBeInTheDocument()
    expect(screen.getByText('blocked')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create staged rollout' })).toBeDisabled()
    expect(screen.getByLabelText('Failure threshold %')).toBeInTheDocument()
    expect(screen.getByLabelText('Soak target (s)')).toBeInTheDocument()
    expect(screen.getByLabelText('ACK timeout (s)')).toHaveValue(900)
  })

  it('enforces fleet hardware policy in the UI and includes the ACK timeout on create', async () => {
    createAgentRollout.mockResolvedValue({ id: 'new-rollout' })
    fetchAgentRollout.mockResolvedValue({ id: 'new-rollout', name: 'Canary rollout', status: 'draft', batch_size: 1, failure_threshold_percent: 20, soak_seconds: 300, ack_timeout_seconds: 900, targets: [] })
    render(<I18nProvider><FleetRolloutPanel agents={[compatibleAgent, changedAgent]} canManage /></I18nProvider>)

    const goodLabel = (await screen.findByText('station-good')).closest('label')
    const changedLabel = screen.getByText('station-changed').closest('label')
    expect(within(goodLabel).getByRole('checkbox')).toBeEnabled()
    expect(within(changedLabel).getByRole('checkbox')).toBeDisabled()
    expect(screen.getByText('Fleet policy · 1/2 eligible')).toBeInTheDocument()

    fireEvent.click(within(goodLabel).getByRole('checkbox'))
    fireEvent.click(screen.getByRole('button', { name: 'Create staged rollout' }))

    await vi.waitFor(() => expect(createAgentRollout).toHaveBeenCalledWith(expect.objectContaining({
      agent_ids: ['station-good'],
      ack_timeout_seconds: 900,
    })))
  
    await flushReactUpdates()})

  it('shows cohort progress and blocks promotion while the active batch is unacknowledged', async () => {
    fetchAgentRollouts.mockResolvedValue([{ id: 'rollout-1', name: 'Canary alpha', status: 'running' }])
    fetchAgentRollout.mockResolvedValue(running)
    render(<I18nProvider><FleetRolloutPanel agents={[]} canManage /></I18nProvider>)
    fireEvent.click(await screen.findByRole('button', { name: 'Canary alpha' }))
    expect(await screen.findByText('2/4 · 50%')).toBeInTheDocument()
    expect(screen.getByText('#2')).toBeInTheDocument()
    expect(screen.getByLabelText('Soak timer')).toBeInTheDocument()
    expect(screen.getByLabelText('ACK deadline')).toBeInTheDocument()
    expect(screen.getByText('cohort 2 · config v4')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Advance batch' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Advance batch' })).toHaveAttribute('title', 'Wait for the active batch to acknowledge')
  })

  it('matches the backend rejection threshold denominator using decided targets only', async () => {
    const thresholdCase = {
      ...running,
      targets: [
        { agent_id: 'applied', ordinal: 0, status: 'applied' },
        { agent_id: 'rejected', ordinal: 1, status: 'rejected' },
        ...Array.from({ length: 8 }, (_, index) => ({ agent_id: `pending-${index}`, ordinal: index + 2, status: 'pending' })),
      ],
    }
    fetchAgentRollouts.mockResolvedValue([{ id: 'rollout-1', name: 'Canary alpha', status: 'running' }])
    fetchAgentRollout.mockResolvedValue(thresholdCase)
    render(<I18nProvider><FleetRolloutPanel agents={[]} canManage /></I18nProvider>)
    fireEvent.click(await screen.findByRole('button', { name: 'Canary alpha' }))
    expect(await screen.findByText('1/2 · 50%')).toBeInTheDocument()
    expect(screen.getByText(/rejection rate exceeds/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Advance batch' })).toBeDisabled()
  })

  it('lets the operator reconcile an overdue batch to failed from the UI', async () => {
    const overdue = {
      ...running,
      ack_timeout_seconds: 30,
      targets: running.targets.map((target, index) => index === 2 ? { ...target, issued_at: new Date(Date.now() - 60000).toISOString() } : target),
    }
    const failed = { ...overdue, status: 'failed' }
    fetchAgentRollouts.mockResolvedValue([{ id: 'rollout-1', name: 'Canary alpha', status: 'running' }])
    fetchAgentRollout.mockResolvedValueOnce(overdue).mockResolvedValueOnce(failed)
    agentRolloutAction.mockRejectedValue(new Error('rollout acknowledgement timeout exceeded'))

    render(<I18nProvider><FleetRolloutPanel agents={[]} canManage /></I18nProvider>)
    fireEvent.click(await screen.findByRole('button', { name: 'Canary alpha' }))
    const failButton = await screen.findByRole('button', { name: 'Fail overdue batch' })
    expect(screen.getByText('Expired · fail closed')).toBeInTheDocument()
    fireEvent.click(failButton)

    await vi.waitFor(() => expect(agentRolloutAction).toHaveBeenCalledWith('rollout-1', 'advance'))
    expect(await screen.findByText('rollout acknowledgement timeout exceeded')).toBeInTheDocument()
    await vi.waitFor(() => expect(screen.getByText('failed')).toBeInTheDocument())
  
    await flushReactUpdates()
})

  it('requires confirmation before rollback', async () => {
    fetchAgentRollouts.mockResolvedValue([{ id: 'rollout-1', name: 'Canary alpha', status: 'running' }])
    fetchAgentRollout.mockResolvedValue({ ...running, targets: running.targets.map((target) => ({ ...target, status: 'applied' })) })
    render(<I18nProvider><FleetRolloutPanel agents={[]} canManage /></I18nProvider>)
    fireEvent.click(await screen.findByRole('button', { name: 'Canary alpha' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Rollback' }))

    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText(/Canary alpha/)).toBeInTheDocument()
    fireEvent.click(within(dialog).getByRole('button', { name: /^cancel$/i }))

    expect(agentRolloutAction).not.toHaveBeenCalled()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('rolls back only after the dialog is confirmed', async () => {
    fetchAgentRollouts.mockResolvedValue([{ id: 'rollout-1', name: 'Canary alpha', status: 'running' }])
    fetchAgentRollout.mockResolvedValue({ ...running, targets: running.targets.map((target) => ({ ...target, status: 'applied' })) })
    agentRolloutAction.mockResolvedValue({ ...running, status: 'rolled_back' })
    render(<I18nProvider><FleetRolloutPanel agents={[]} canManage /></I18nProvider>)
    fireEvent.click(await screen.findByRole('button', { name: 'Canary alpha' }))
    fireEvent.click(await screen.findByRole('button', { name: 'Rollback' }))

    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^rollback$/i }))

    await vi.waitFor(() => expect(agentRolloutAction).toHaveBeenCalledWith('rollout-1', 'rollback'))
    await vi.waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument())
  
    await flushReactUpdates()
})
})
