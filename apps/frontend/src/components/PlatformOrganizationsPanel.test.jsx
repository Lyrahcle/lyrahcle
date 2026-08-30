import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import PlatformOrganizationsPanel from './PlatformOrganizationsPanel.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({
  createOrganization: vi.fn(),
  updateOrganization: vi.fn(),
  grantPlatformAdministrator: vi.fn(),
  revokePlatformAdministrator: vi.fn(),
}))

describe('PlatformOrganizationsPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('lists organizations with status and resolves organization names for platform users', () => {
    render(
      <I18nProvider>
        <PlatformOrganizationsPanel
          organizations={[{ id: 'org-1', slug: 'mission-a', name: 'Mission A', enabled: true }]}
          platformUsers={[{ id: 'user-1', organization_id: 'org-1', email: 'ops@example.com', enabled: true, platform_admin: false }]}
          onChanged={() => {}}
        />
      </I18nProvider>,
    )
    const [organizationsTable, platformUsersTable] = screen.getAllByRole('table')
    expect(within(organizationsTable).getByText('Mission A')).toBeInTheDocument()
    expect(within(organizationsTable).getByText('mission-a')).toBeInTheDocument()
    expect(within(organizationsTable).getByText('Enabled')).toBeInTheDocument()
    expect(within(platformUsersTable).getAllByText('ops@example.com')).toHaveLength(2)
    expect(within(platformUsersTable).getByText('Mission A')).toBeInTheDocument()
    expect(within(platformUsersTable).getByText('mission-a')).toBeInTheDocument()
  })

  it('creates an organization from the UI and refreshes the list', async () => {
    api.createOrganization.mockResolvedValue({ id: 'org-2' })
    const onChanged = vi.fn().mockResolvedValue()
    render(<I18nProvider><PlatformOrganizationsPanel organizations={[]} onChanged={onChanged} /></I18nProvider>)

    fireEvent.change(screen.getByLabelText(/^slug$/i), { target: { value: 'MISSION-B' } })
    fireEvent.change(screen.getByLabelText(/^name$/i), { target: { value: ' Mission B ' } })
    fireEvent.click(screen.getByRole('button', { name: /create organization/i }))

    await waitFor(() => expect(api.createOrganization).toHaveBeenCalledWith('mission-b', 'Mission B'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('edits an organization from the platform UI and refreshes the list', async () => {
    api.updateOrganization.mockResolvedValue({ id: 'org-1', slug: 'mission-renamed', name: 'Mission Renamed' })
    const onChanged = vi.fn().mockResolvedValue()
    render(
      <I18nProvider>
        <PlatformOrganizationsPanel
          organizations={[{ id: 'org-1', slug: 'mission-a', name: 'Mission A', enabled: true }]}
          onChanged={onChanged}
        />
      </I18nProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: /^edit$/i }))
    fireEvent.change(screen.getByLabelText(/organization slug/i), { target: { value: 'MISSION-RENAMED' } })
    fireEvent.change(screen.getByLabelText(/organization name/i), { target: { value: ' Mission Renamed ' } })
    fireEvent.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => expect(api.updateOrganization).toHaveBeenCalledWith('org-1', 'mission-renamed', 'Mission Renamed'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('grants platform administrator access from the UI', async () => {
    api.grantPlatformAdministrator.mockResolvedValue()
    const onPlatformUsersChanged = vi.fn().mockResolvedValue()
    render(
      <I18nProvider>
        <PlatformOrganizationsPanel
          organizations={[{ id: 'org-1', slug: 'mission-a', name: 'Mission A', enabled: true }]}
          platformUsers={[{ id: 'user-1', organization_id: 'org-1', email: 'ops@example.com', enabled: true, platform_admin: false }]}
          onChanged={() => {}}
          onPlatformUsersChanged={onPlatformUsersChanged}
        />
      </I18nProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: /grant platform admin/i }))
    await waitFor(() => expect(api.grantPlatformAdministrator).toHaveBeenCalledWith('user-1'))
    expect(onPlatformUsersChanged).toHaveBeenCalledOnce()
  })

  it('surfaces creation failures', async () => {
    api.createOrganization.mockRejectedValue(new Error('organization slug already registered'))
    render(<I18nProvider><PlatformOrganizationsPanel organizations={[]} onChanged={() => {}} /></I18nProvider>)

    fireEvent.change(screen.getByLabelText(/^slug$/i), { target: { value: 'mission-b' } })
    fireEvent.change(screen.getByLabelText(/^name$/i), { target: { value: 'Mission B' } })
    fireEvent.click(screen.getByRole('button', { name: /create organization/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent('organization slug already registered')
  })
})
