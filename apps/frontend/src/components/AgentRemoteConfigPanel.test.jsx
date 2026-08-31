import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import AgentRemoteConfigPanel from './AgentRemoteConfigPanel.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({
  applyAgentConfig: vi.fn(),
  fetchAgentConfig: vi.fn(),
  fetchAgentConfigHistory: vi.fn(),
  rollbackAgentConfig: vi.fn(),
  fetchAgentRollouts: vi.fn(),
  fetchAgentRollout: vi.fn(),
  createAgentRollout: vi.fn(),
  agentRolloutAction: vi.fn(),
}))

function renderPanel() {
  return render(
    <I18nProvider>
      <AgentRemoteConfigPanel
        agents={[{ agent_id: 'sdr-1', status: 'active' }]}
        satellites={[{ id: 'sat-1', name: 'ISS' }]}
      />
    </I18nProvider>,
  )
}

describe('AgentRemoteConfigPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.fetchAgentConfig.mockResolvedValue(null)
    api.fetchAgentConfigHistory.mockResolvedValue([])
    api.fetchAgentRollouts.mockResolvedValue([])
  })

  it('loads desired config and history for the active agent', async () => {
    api.fetchAgentConfig.mockResolvedValue({
      version: 4,
      ack_status: 'applied',
      config: { frequency_hz: 145800000, capture_interval_seconds: 5, satellite_id: 'sat-1' },
    })
    api.fetchAgentConfigHistory.mockResolvedValue([])
    renderPanel()

    await waitFor(() => expect(api.fetchAgentConfig).toHaveBeenCalledWith('sdr-1'))
    expect(screen.getByDisplayValue('145800000')).toBeInTheDocument()
    expect(screen.getByDisplayValue('5')).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: /target satellite/i })).toHaveValue('sat-1')
  })

  it('applies numeric remote configuration and reloads state', async () => {
    api.applyAgentConfig.mockResolvedValue({})
    renderPanel()
    await waitFor(() => expect(api.fetchAgentConfig).toHaveBeenCalled())

    const numeric = screen.getAllByRole('spinbutton')
    fireEvent.change(numeric[0], { target: { value: '437500000' } })
    fireEvent.change(numeric[1], { target: { value: '12' } })
    fireEvent.change(screen.getAllByRole('combobox')[1], { target: { value: 'sat-1' } })
    fireEvent.click(screen.getByRole('button', { name: /publish config/i }))

    await waitFor(() => expect(api.applyAgentConfig).toHaveBeenCalledWith('sdr-1', 437500000, 12, 'sat-1'))
    expect(api.fetchAgentConfig.mock.calls.length).toBeGreaterThanOrEqual(2)
    expect(api.fetchAgentConfigHistory.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('shows the signing KID for desired and historical configuration versions', async () => {
    const kid = 'ed25519-sha256-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef'
    api.fetchAgentConfig.mockResolvedValue({
      id: 'cfg-current',
      version: 5,
      signing_kid: kid,
      ack_status: 'applied',
      config: { frequency_hz: 145800000, capture_interval_seconds: 5, satellite_id: 'sat-1' },
    })
    api.fetchAgentConfigHistory.mockResolvedValue([{
      id: 'cfg-history',
      version: 4,
      signing_kid: kid,
      ack_status: 'applied',
      config: { frequency_hz: 145800000, capture_interval_seconds: 5, satellite_id: 'sat-1' },
    }])
    renderPanel()

    await waitFor(() => expect(api.fetchAgentConfigHistory).toHaveBeenCalledWith('sdr-1'))
    expect(screen.getAllByText(/signing key/i).length).toBeGreaterThanOrEqual(2)
    expect(screen.getAllByTitle(kid)).toHaveLength(2)
  })

  it('surfaces remote configuration failures', async () => {
    api.applyAgentConfig.mockRejectedValue(new Error('config rejected'))
    renderPanel()
    await waitFor(() => expect(api.fetchAgentConfig).toHaveBeenCalled())

    fireEvent.click(screen.getByRole('button', { name: /publish config/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('config rejected')
  })
})
