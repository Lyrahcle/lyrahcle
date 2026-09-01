import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import { clearEffectivePermissions, setEffectivePermissions } from '../lib/permissions'
import DetectionRulesPanel from './DetectionRulesPanel.jsx'
import * as api from '../lib/api'

vi.mock('../lib/api', () => ({
  createDetectionRule: vi.fn(),
  updateDetectionRule: vi.fn(),
}))

function renderPanel(page = { items: [] }, onChanged = vi.fn()) {
  return render(
    <I18nProvider>
      <DetectionRulesPanel page={page} onChanged={onChanged} />
    </I18nProvider>,
  )
}

describe('DetectionRulesPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearEffectivePermissions()
  })

  it('hides mutation controls without detections.manage', () => {
    setEffectivePermissions(['detections.read'])
    renderPanel({
      items: [{
        id: 'rule-1', name: 'High RF deviation', metric: 'power_dbm',
        operator: 'abs_z_gte', threshold: 6, severity: 'high',
        cooldown_seconds: 120, auto_incident: true, enabled: true,
      }],
    })
    expect(screen.queryByRole('button', { name: /create rule/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /^enabled$/i })).not.toBeInTheDocument()
    expect(screen.getByText('High RF deviation')).toBeInTheDocument()
  })

  it('creates a rule for a custom role with detections.manage', async () => {
    setEffectivePermissions(['detections.read', 'detections.manage'])
    api.createDetectionRule.mockResolvedValue({})
    const onChanged = vi.fn()
    renderPanel({ items: [] }, onChanged)

    fireEvent.change(screen.getByPlaceholderText(/rule name/i), { target: { value: 'Critical burst' } })
    fireEvent.click(screen.getByRole('button', { name: /create rule/i }))

    await waitFor(() => expect(api.createDetectionRule).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Critical burst',
      threshold: 6,
      cooldownSeconds: 120,
      autoIncident: true,
    })))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('toggles an existing rule when management permission is present', async () => {
    setEffectivePermissions(['detections.manage'])
    api.updateDetectionRule.mockResolvedValue({})
    const onChanged = vi.fn()
    renderPanel({
      items: [{
        id: 'rule-1', name: 'High RF deviation', metric: 'power_dbm',
        operator: 'abs_z_gte', threshold: 6, severity: 'high',
        cooldown_seconds: 120, auto_incident: true, enabled: true,
      }],
    }, onChanged)

    fireEvent.click(screen.getByRole('button', { name: /^enabled$/i }))
    await waitFor(() => expect(api.updateDetectionRule).toHaveBeenCalledWith('rule-1', { enabled: false }))
    expect(onChanged).toHaveBeenCalledOnce()
  })
})
