import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import ConnectivityOperationsOverview from './ConnectivityOperationsOverview.jsx'

function renderOverview(identities, keys = []) {
  return render(<I18nProvider><ConnectivityOperationsOverview agentIdentities={identities} agentKeys={keys} /></I18nProvider>)
}

describe('ConnectivityOperationsOverview', () => {
  it('surfaces healthy, attention and offline agent posture in one view', () => {
    const now = Date.now()
    renderOverview([
      { agent_id: 'sdr-healthy', status: 'active', last_seen_at: new Date(now - 60_000).toISOString(), device: { hardware: 'RTL-SDR', compatibility: 'compatible', capabilities: { integrity_status: 'healthy' } } },
      { agent_id: 'sdr-tampered', status: 'active', last_seen_at: new Date(now - 60_000).toISOString(), device: { hardware: 'HackRF', compatibility: 'compatible', capabilities: { integrity_status: 'changed' } } },
      { agent_id: 'sdr-offline', status: 'active', last_seen_at: new Date(now - 3_600_000).toISOString() },
    ])

    expect(screen.getByRole('heading', { name: /connectivity operations/i })).toBeInTheDocument()
    expect(screen.getByText('sdr-healthy')).toBeInTheDocument()
    expect(screen.getByText('sdr-tampered')).toBeInTheDocument()
    expect(screen.getByText('sdr-offline')).toBeInTheDocument()
    expect(screen.getAllByText('1')).toHaveLength(3)
  })

  it('filters attention agents and supports agent-id search', () => {
    const now = Date.now()
    renderOverview([
      { agent_id: 'madrid-edge', status: 'active', last_seen_at: new Date(now - 30_000).toISOString(), device: { compatibility: 'degraded', capabilities: { integrity_status: 'healthy' } } },
      { agent_id: 'barcelona-edge', status: 'active', last_seen_at: new Date(now - 30_000).toISOString(), device: { compatibility: 'compatible', capabilities: { integrity_status: 'healthy' } } },
    ])

    fireEvent.click(screen.getByRole('button', { name: /needs attention/i }))
    expect(screen.getByText('madrid-edge')).toBeInTheDocument()
    expect(screen.queryByText('barcelona-edge')).not.toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText(/search agent id/i), { target: { value: 'no-match' } })
    expect(screen.getByText(/no agents match/i)).toBeInTheDocument()
  })
})
