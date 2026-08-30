import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import { clearEffectivePermissions, setEffectivePermissions } from '../lib/permissions.js'
import AssetAlertsPanel from './AssetAlertsPanel.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({ acknowledgeAssetAlert: vi.fn() }))

const page = {
  items: [{
    id: 'alert-1',
    asset_value: 'ops.example.com',
    asset_kind: 'domain',
    cve_id: 'CVE-2026-1234',
    severity: 'high',
    matched_keyword: 'satellite',
    acknowledged: false,
  }],
  total: 1,
  limit: 25,
  offset: 0,
}

function renderPanel(onChanged = vi.fn()) {
  return render(
    <I18nProvider>
      <AssetAlertsPanel page={page} onPrev={() => {}} onNext={() => {}} onChanged={onChanged} />
    </I18nProvider>,
  )
}

describe('AssetAlertsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearEffectivePermissions()
  })

  it('shows alerts but not acknowledgement controls without alerts.manage', () => {
    setEffectivePermissions(['alerts.read'])
    renderPanel()
    expect(screen.getByText('CVE-2026-1234')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /acknowledge/i })).not.toBeInTheDocument()
  })

  it('acknowledges an alert with alerts.manage', async () => {
    setEffectivePermissions(['alerts.read', 'alerts.manage'])
    api.acknowledgeAssetAlert.mockResolvedValue({})
    const onChanged = vi.fn()
    renderPanel(onChanged)

    fireEvent.click(screen.getByRole('button', { name: /acknowledge/i }))
    await waitFor(() => expect(api.acknowledgeAssetAlert).toHaveBeenCalledWith('alert-1'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('surfaces acknowledgement API errors', async () => {
    setEffectivePermissions(['alerts.manage'])
    api.acknowledgeAssetAlert.mockRejectedValue(new Error('ack failed'))
    renderPanel()

    fireEvent.click(screen.getByRole('button', { name: /acknowledge/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('ack failed')
  })
})
