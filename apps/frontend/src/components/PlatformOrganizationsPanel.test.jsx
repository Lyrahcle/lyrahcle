import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import PlatformOrganizationsPanel from './PlatformOrganizationsPanel.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({ createOrganization: vi.fn() }))

describe('PlatformOrganizationsPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('lists organizations returned by the platform API', () => {
    render(<I18nProvider><PlatformOrganizationsPanel organizations={[{ id: 'org-1', slug: 'mission-a', name: 'Mission A' }]} onChanged={() => {}} /></I18nProvider>)
    expect(screen.getByText('Mission A')).toBeInTheDocument()
    expect(screen.getByText('mission-a')).toBeInTheDocument()
  })

  it('creates an organization from the UI and refreshes the list', async () => {
    api.createOrganization.mockResolvedValue({ id: 'org-2' })
    const onChanged = vi.fn().mockResolvedValue()
    render(<I18nProvider><PlatformOrganizationsPanel organizations={[]} onChanged={onChanged} /></I18nProvider>)

    fireEvent.change(screen.getByLabelText(/slug/i), { target: { value: 'MISSION-B' } })
    fireEvent.change(screen.getByLabelText(/^name$/i), { target: { value: ' Mission B ' } })
    fireEvent.click(screen.getByRole('button', { name: /create organization/i }))

    await waitFor(() => expect(api.createOrganization).toHaveBeenCalledWith('mission-b', 'Mission B'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('surfaces creation failures', async () => {
    api.createOrganization.mockRejectedValue(new Error('organization slug already registered'))
    render(<I18nProvider><PlatformOrganizationsPanel organizations={[]} onChanged={() => {}} /></I18nProvider>)

    fireEvent.change(screen.getByLabelText(/slug/i), { target: { value: 'mission-b' } })
    fireEvent.change(screen.getByLabelText(/^name$/i), { target: { value: 'Mission B' } })
    fireEvent.click(screen.getByRole('button', { name: /create organization/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent('organization slug already registered')
  })
})
