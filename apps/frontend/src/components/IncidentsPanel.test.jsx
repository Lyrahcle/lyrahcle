import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import { clearEffectivePermissions, setEffectivePermissions } from '../lib/permissions'
import IncidentsPanel from './IncidentsPanel.jsx'
import * as api from '../lib/api'

vi.mock('../lib/api', () => ({
  addIncidentEvidence: vi.fn(),
  addIncidentNote: vi.fn(),
  createIncident: vi.fn(),
  fetchIncident: vi.fn(),
  updateIncident: vi.fn(),
}))

const filters = { status: '', severity: '', agentId: '', satelliteId: '' }

function renderPanel(props = {}) {
  return render(
    <I18nProvider>
      <IncidentsPanel
        page={{ items: [], total: 0, limit: 25, offset: 0 }}
        filters={filters}
        onFiltersChange={() => {}}
        onPrev={() => {}}
        onNext={() => {}}
        onChanged={() => {}}
        {...props}
      />
    </I18nProvider>,
  )
}

describe('IncidentsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearEffectivePermissions()
  })

  it('is read-only without incidents.manage', () => {
    setEffectivePermissions(['incidents.read'])
    renderPanel()
    expect(screen.queryByRole('button', { name: /create incident/i })).not.toBeInTheDocument()
    expect(screen.getByText(/no incidents match/i)).toBeInTheDocument()
  })

  it('summarizes incident posture and applies quick filters', () => {
    setEffectivePermissions(['incidents.read'])
    const onFiltersChange = vi.fn()
    const items = [
      { id: 'i1', reference: 'INC-1', title: 'Critical RF', severity: 'critical', status: 'new', source_type: 'rf', updated_at: '2026-08-31T12:00:00Z', assigned_to: '' },
      { id: 'i2', reference: 'INC-2', title: 'Contained issue', severity: 'medium', status: 'contained', source_type: 'agent', updated_at: '2026-08-31T12:05:00Z', assigned_to: 'user-1' },
    ]
    renderPanel({ page: { items, total: 7, limit: 25, offset: 0 }, onFiltersChange })

    expect(screen.getByRole('heading', { name: /incident posture/i })).toBeInTheDocument()
    expect(screen.getByText('7')).toBeInTheDocument()
    expect(screen.getAllByText('1')).toHaveLength(2)
    fireEvent.click(screen.getByRole('button', { name: /critical only/i }))
    expect(onFiltersChange).toHaveBeenCalledWith(expect.objectContaining({ severity: 'critical' }))
  })

  it('creates incidents for a custom role with incidents.manage', async () => {
    setEffectivePermissions(['incidents.read', 'incidents.manage'])
    api.createIncident.mockResolvedValue({})
    const onChanged = vi.fn()
    renderPanel({ onChanged })

    fireEvent.change(screen.getByLabelText(/^Title$/i), { target: { value: 'Unexpected RF burst' } })
    fireEvent.click(screen.getByRole('button', { name: /create incident/i }))

    await waitFor(() => expect(api.createIncident).toHaveBeenCalledWith(expect.objectContaining({
      title: 'Unexpected RF burst',
      severity: 'medium',
      assignedTo: '',
    })))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('exposes assignee controls with incidents.manage plus users.read', () => {
    setEffectivePermissions(['incidents.manage'])
    const users = [{ id: 'user-1', username: 'analyst-one', email: 'a@example.com' }]
    const { rerender } = renderPanel({ users })
    expect(screen.queryByRole('option', { name: 'analyst-one' })).not.toBeInTheDocument()

    setEffectivePermissions(['incidents.manage', 'users.read'])
    rerender(
      <I18nProvider>
        <IncidentsPanel
          page={{ items: [], total: 0, limit: 25, offset: 0 }}
          filters={filters}
          onFiltersChange={() => {}}
          onPrev={() => {}}
          onNext={() => {}}
          onChanged={() => {}}
          users={users}
        />
      </I18nProvider>,
    )
    expect(screen.getByRole('option', { name: 'analyst-one' })).toBeInTheDocument()
  })
})
