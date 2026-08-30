import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import AgentIdentityPanel from './AgentIdentityPanel.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({
  createAgentEnrollmentToken: vi.fn(),
  updateAgentIdentityStatus: vi.fn(),
  updateAgentStation: vi.fn(),
}))

function renderPanel(agents = [], onChanged = vi.fn()) {
  return render(
    <I18nProvider>
      <AgentIdentityPanel agents={agents} onChanged={onChanged} />
    </I18nProvider>,
  )
}

describe('AgentIdentityPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('issues an enrollment token once and refreshes identities', async () => {
    api.createAgentEnrollmentToken.mockResolvedValue({
      agent_id: 'sdr-madrid',
      token: 'enroll-secret',
      expires_at: '2026-08-30T13:00:00Z',
    })
    const onChanged = vi.fn().mockResolvedValue()
    renderPanel([], onChanged)

    fireEvent.change(screen.getByLabelText(/agent id/i), { target: { value: ' sdr-madrid ' } })
    fireEvent.click(screen.getByRole('button', { name: /issue enrollment token/i }))

    await waitFor(() => expect(api.createAgentEnrollmentToken).toHaveBeenCalledWith('sdr-madrid'))
    expect(await screen.findByRole('status')).toHaveTextContent('enroll-secret')
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('quarantines an active mTLS identity', async () => {
    api.updateAgentIdentityStatus.mockResolvedValue({})
    const onChanged = vi.fn().mockResolvedValue()
    renderPanel([{
      agent_id: 'sdr-1',
      status: 'active',
      certificate: null,
      last_seen_at: null,
    }], onChanged)

    fireEvent.click(screen.getByRole('button', { name: /quarantine/i }))
    await waitFor(() => expect(api.updateAgentIdentityStatus).toHaveBeenCalledWith('sdr-1', 'quarantined'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('updates station coordinates as numbers', async () => {
    api.updateAgentStation.mockResolvedValue({})
    const onChanged = vi.fn().mockResolvedValue()
    renderPanel([{
      agent_id: 'sdr-1',
      status: 'active',
      certificate: null,
      last_seen_at: null,
      station: null,
    }], onChanged)

    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'sdr-1' } })
    const numeric = screen.getAllByRole('spinbutton')
    fireEvent.change(numeric[0], { target: { value: '40.4168' } })
    fireEvent.change(numeric[1], { target: { value: '-3.7038' } })
    fireEvent.change(numeric[2], { target: { value: '667' } })
    fireEvent.click(screen.getByRole('button', { name: /save station/i }))

    await waitFor(() => expect(api.updateAgentStation).toHaveBeenCalledWith('sdr-1', 40.4168, -3.7038, 667))
    expect(onChanged).toHaveBeenCalledOnce()
  })
})
