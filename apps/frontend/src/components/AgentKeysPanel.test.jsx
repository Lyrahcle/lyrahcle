import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import AgentKeysPanel from './AgentKeysPanel.jsx'
import * as api from '../lib/api.js'

function renderPanel(props) {
  return render(
    <I18nProvider>
      <AgentKeysPanel {...props} />
    </I18nProvider>,
  )
}

vi.mock('../lib/api.js', () => ({
  createAgentKey: vi.fn(),
  revokeAgentKey: vi.fn(),
}))

describe('AgentKeysPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('issues a one-time agent key and refreshes the list', async () => {
    api.createAgentKey.mockResolvedValue({ agent_id: 'sdr-madrid', api_key: 'secret-once' })
    const onChanged = vi.fn()
    renderPanel({ keys: [], onChanged })

    fireEvent.change(screen.getByLabelText(/new agent_id/i), { target: { value: '  sdr-madrid  ' } })
    fireEvent.click(screen.getByRole('button', { name: /issue key/i }))

    await waitFor(() => expect(api.createAgentKey).toHaveBeenCalledWith('sdr-madrid'))
    expect(await screen.findByRole('status')).toHaveTextContent('secret-once')
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('revokes an active legacy key', async () => {
    api.revokeAgentKey.mockResolvedValue({})
    const onChanged = vi.fn()
    renderPanel({
      keys: [{
        agent_id: 'sdr-1',
        created_at: '2026-08-01T00:00:00Z',
        last_used_at: null,
        revoked: false,
      }],
      onChanged,
    })

    fireEvent.click(screen.getByRole('button', { name: /revoke key for sdr-1/i }))
    await waitFor(() => expect(api.revokeAgentKey).toHaveBeenCalledWith('sdr-1'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('surfaces key issuance failures', async () => {
    api.createAgentKey.mockRejectedValue(new Error('key issuance failed'))
    renderPanel({ keys: [], onChanged: () => {} })

    fireEvent.change(screen.getByLabelText(/new agent_id/i), { target: { value: 'sdr-1' } })
    fireEvent.click(screen.getByRole('button', { name: /issue key/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent('key issuance failed')
  })
})
