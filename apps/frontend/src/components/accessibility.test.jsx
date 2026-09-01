import { describe, expect, test, vi } from 'vitest'
import { render, screen, within, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { axe } from 'jest-axe'
import { I18nProvider } from '../lib/I18nContext.jsx'
import NewAssetForm from './NewAssetForm.jsx'
import NewSatelliteForm from './NewSatelliteForm.jsx'
import ChangePasswordModal from './ChangePasswordModal.jsx'
import SettingsPanel from './SettingsPanel.jsx'
import Login from './Login.jsx'
import IncidentsPanel from './IncidentsPanel.jsx'

const flushReactUpdates = async () => {
  await act(async () => {
    await Promise.resolve()
    await Promise.resolve()
  })
}

vi.mock('../lib/api.js', () => ({
  fetchPrograms: vi.fn().mockResolvedValue([]),
  createProgram: vi.fn(),
  createAsset: vi.fn(),
  createSatellite: vi.fn(),
  changePassword: vi.fn(),
  fetchIncident: vi.fn().mockResolvedValue({ incident: null, events: [] }),
  createIncident: vi.fn(),
  updateIncident: vi.fn(),
  addIncidentNote: vi.fn(),
  fetchSystemInfo: vi.fn().mockResolvedValue({
    environment: 'test',
    license: 'AGPLv3 (core)',
    celestrak_groups: ['stations'],
    celestrak_sync_schedule: 'every 6 hours',
    nvd_sync_note: 'managed by the cve-sync worker',
  }),
}))

function withI18n(ui) {
  return <I18nProvider>{ui}</I18nProvider>
}

describe('form label associations', () => {
  test('every NewAssetForm field is reachable by its visible label', async () => {
    render(withI18n(<NewAssetForm onCreated={() => {}} />))
    await userEvent.click(screen.getByRole('button', { name: /new asset/i }))

    expect(screen.getByLabelText('Program')).toBeInTheDocument()
    expect(screen.getByLabelText('Kind')).toBeInTheDocument()
    expect(screen.getByLabelText('Value')).toBeInTheDocument()
  })

  test('every NewSatelliteForm field is reachable by its visible label', async () => {
    render(withI18n(<NewSatelliteForm onCreated={() => {}} />))
    await userEvent.click(screen.getByRole('button', { name: /register satellite/i }))

    expect(screen.getByLabelText('NORAD ID')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toBeInTheDocument()
    expect(screen.getByLabelText('TLE line 1')).toBeInTheDocument()
    expect(screen.getByLabelText('TLE line 2')).toBeInTheDocument()
  })
})

describe('ChangePasswordModal dialog semantics', () => {
  test('exposes role=dialog, aria-modal, and an accessible name', () => {
    render(withI18n(<ChangePasswordModal onClose={() => {}} />))
    const dialog = screen.getByRole('dialog', { name: /change password/i })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
  })

  test('moves focus into the dialog on open', () => {
    render(withI18n(<ChangePasswordModal onClose={() => {}} />))
    const dialog = screen.getByRole('dialog')
    expect(dialog.contains(document.activeElement)).toBe(true)
  })

  test('Tab cycles focus without leaving the dialog', async () => {
    render(withI18n(<ChangePasswordModal onClose={() => {}} />))
    const dialog = screen.getByRole('dialog')
    const focusables = within(dialog).getAllByRole('button').concat(within(dialog).getAllByLabelText(/password/i))

    
    for (let i = 0; i < focusables.length + 2; i++) {
      await userEvent.tab()
    }
    expect(dialog.contains(document.activeElement)).toBe(true)
  })

  test('Escape calls onClose', async () => {
    const onClose = vi.fn()
    render(withI18n(<ChangePasswordModal onClose={onClose} />))
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })

  test('returns focus to the previously focused element on close', () => {
    const trigger = document.createElement('button')
    document.body.appendChild(trigger)
    trigger.focus()

    const { unmount } = render(withI18n(<ChangePasswordModal onClose={() => {}} />))
    expect(document.activeElement).not.toBe(trigger)

    unmount()
    expect(document.activeElement).toBe(trigger)
    document.body.removeChild(trigger)
  })
})

describe('SettingsPanel', () => {
  test('theme and language options are grouped with accessible names', async () => {
    render(withI18n(<SettingsPanel />))
    expect(screen.getByRole('group', { name: 'Appearance' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Language' })).toBeInTheDocument()
  
    await flushReactUpdates()})

  test('selected option is marked aria-pressed', async () => {
    render(withI18n(<SettingsPanel />))
    const darkButton = screen.getByRole('button', { name: /dark/i })
    const lightButton = screen.getByRole('button', { name: /light/i })
    expect([darkButton.getAttribute('aria-pressed'), lightButton.getAttribute('aria-pressed')]).toContain('true')
  
    await flushReactUpdates()})
})

describe('automated accessibility audit (axe)', () => {
  test('Login has no detectable WCAG violations', async () => {
    const { container } = render(withI18n(<Login onSuccess={() => {}} />))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  test('SettingsPanel has no detectable WCAG violations', async () => {
    const { container } = render(withI18n(<SettingsPanel />))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  
    await flushReactUpdates()})

  test('IncidentsPanel empty state has no detectable WCAG violations', async () => {
    const { container } = render(withI18n(
      <IncidentsPanel
        page={{ items: [], total: 0, limit: 50, offset: 0 }}
        filters={{ status: '', severity: '', agentId: '' }}
        onFiltersChange={() => {}}
        onPrev={() => {}}
        onNext={() => {}}
        onChanged={() => {}}
      />,
    ))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })

  test('an open ChangePasswordModal has no detectable WCAG violations', async () => {
    const { container } = render(withI18n(<ChangePasswordModal onClose={() => {}} />))
    const results = await axe(container)
    expect(results).toHaveNoViolations()
  })
})
