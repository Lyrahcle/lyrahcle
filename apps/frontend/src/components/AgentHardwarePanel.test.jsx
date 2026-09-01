import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import AgentHardwarePanel from './AgentHardwarePanel.jsx'
import { I18nProvider } from '../lib/I18nContext'

const agent = {
  agent_id: 'edge-madrid-01', status: 'active',
  device: {
    driver: 'rtlsdr', hardware: 'RTL-SDR Blog V4', manufacturer: 'RTL-SDR Blog', product: 'V4', serial: '000001', rx: true, tx: false,
    compatibility: 'compatible', compatibility_reason: 'software capability report satisfies the v1 baseline; physical lab validation is tracked separately',
    reported_at: '2026-08-31T04:00:00Z',
    capabilities: {
      architecture: 'arm64', os: 'linux', driver_version: '0.6.0', firmware_version: 'n/a',
      frequency_min_hz: 24000000, frequency_max_hz: 1766000000,
      sample_rates_hz: [1024000, 2048000], bandwidths_hz: [1000000, 2000000], gain_elements: ['TUNER'],
      channels: 1, gps: false, pps: true, sample_drops: 3,
    },
  },
}

function renderPanel(props = {}) {
  return render(<I18nProvider><AgentHardwarePanel agents={[agent]} legacyKeys={[]} {...props} /></I18nProvider>)
}

describe('AgentHardwarePanel', () => {
  it('shows tenant agent hardware capabilities and compatibility', () => {
    renderPanel()
    expect(screen.getByText('edge-madrid-01')).toBeInTheDocument()
    expect(screen.getByText('RTL-SDR Blog V4')).toBeInTheDocument()
    expect(screen.getAllByText(/compatible/i).length).toBeGreaterThan(0)
    expect(screen.getByText(/physical lab validation/i)).toBeInTheDocument()
  })

  it('uses mTLS identity as primary record and legacy key only as fallback', () => {
    renderPanel({ legacyKeys: [{ agent_id: 'edge-madrid-01', device: { hardware: 'Legacy device' } }] })
    expect(screen.queryByText('Legacy device')).not.toBeInTheDocument()
    expect(screen.getByText('RTL-SDR Blog V4')).toBeInTheDocument()
  })
})
