import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import SigningKeysPanel from './SigningKeysPanel.jsx'
import * as api from '../lib/api'

vi.mock('../lib/api', () => ({
  fetchAgentConfigSigningKeys: vi.fn(),
  updateAgentConfigSigningRotationPolicy: vi.fn(),
  prepareAgentConfigSigningKeyRotation: vi.fn(),
  fetchAgentConfigSigningPublicKey: vi.fn(),
  activateAgentConfigSigningKey: vi.fn(),
  revokeAgentConfigSigningKey: vi.fn(),
}))

const active = { kid: 'ed25519-sha256-active', algorithm: 'Ed25519', status: 'active', created_at: '2026-08-30T18:00:00Z' }
const staged = { kid: 'ed25519-sha256-staged', algorithm: 'Ed25519', status: 'staged', created_at: '2026-08-30T19:00:00Z' }
const policy = { automatic_prepare: false, interval_seconds: 7776000, staged_alert_after_seconds: 86400, grace_alert_before_seconds: 86400 }
const health = { rotation_due: false, next_rotation_at: '2026-11-28T18:00:00Z', staged_pending: false, warnings: [] }

function managed(keys = [active], extra = {}) {
  return { provider: 'local-managed', managed: true, grace_period_seconds: 604800, keys, rotation_policy: policy, health, ...extra }
}

describe('SigningKeysPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => vi.restoreAllMocks())

  it('stages a key and displays only its public key material', async () => {
    api.fetchAgentConfigSigningKeys
      .mockResolvedValueOnce(managed([active]))
      .mockResolvedValueOnce(managed([active, staged], { health: { ...health, staged_pending: true } }))
    api.prepareAgentConfigSigningKeyRotation.mockResolvedValue({ key: staged, public_key_pem: '-----BEGIN PUBLIC KEY-----\nabc\n-----END PUBLIC KEY-----\n' })

    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    fireEvent.click(await screen.findByRole('button', { name: /prepare rotation/i }))
    expect(await screen.findByText(/BEGIN PUBLIC KEY/)).toBeInTheDocument()
    expect(screen.queryByText(/PRIVATE KEY/)).not.toBeInTheDocument()
    await waitFor(() => expect(api.prepareAgentConfigSigningKeyRotation).toHaveBeenCalledOnce())
  })

  it('activates a staged key with the configured grace period', async () => {
    api.fetchAgentConfigSigningKeys
      .mockResolvedValueOnce(managed([active, staged], { health: { ...health, staged_pending: true } }))
      .mockResolvedValueOnce(managed([{ ...active, status: 'grace' }, { ...staged, status: 'active' }]))
    api.activateAgentConfigSigningKey.mockResolvedValue({ ...staged, status: 'active' })

    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    const table = await screen.findByRole('table')
    fireEvent.change(screen.getByLabelText(/grace period \(hours\)/i), { target: { value: '24' } })
    fireEvent.click(within(table).getByRole('button', { name: /^activate$/i }))
    expect(api.activateAgentConfigSigningKey).not.toHaveBeenCalled()

    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^activate$/i }))
    await waitFor(() => expect(api.activateAgentConfigSigningKey).toHaveBeenCalledWith(staged.kid, 86400))
  })

  it('does not activate a key when the confirmation is cancelled', async () => {
    api.fetchAgentConfigSigningKeys.mockResolvedValue(managed([active, staged], { health: { ...health, staged_pending: true } }))

    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    const table = await screen.findByRole('table')
    fireEvent.click(within(table).getByRole('button', { name: /^activate$/i }))
    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^cancel$/i }))

    expect(api.activateAgentConfigSigningKey).not.toHaveBeenCalled()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })

  it('revokes a key only after a destructive confirmation', async () => {
    api.fetchAgentConfigSigningKeys
      .mockResolvedValueOnce(managed([active, staged], { health: { ...health, staged_pending: true } }))
      .mockResolvedValueOnce(managed([active]))
    api.revokeAgentConfigSigningKey.mockResolvedValue({ ...staged, status: 'revoked' })

    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    const table = await screen.findByRole('table')
    fireEvent.click(within(table).getByRole('button', { name: /^revoke$/i }))
    expect(api.revokeAgentConfigSigningKey).not.toHaveBeenCalled()

    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^revoke$/i }))
    await waitFor(() => expect(api.revokeAgentConfigSigningKey).toHaveBeenCalledWith(staged.kid))
  })

  it('persists automatic staging policy from the platform UI', async () => {
    api.fetchAgentConfigSigningKeys
      .mockResolvedValueOnce(managed())
      .mockResolvedValueOnce(managed([], { rotation_policy: { ...policy, automatic_prepare: true, interval_seconds: 3888000 } }))
    api.updateAgentConfigSigningRotationPolicy.mockResolvedValue({ ...policy, automatic_prepare: true, interval_seconds: 3888000 })

    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    const automatic = await screen.findByRole('checkbox', { name: /automatically prepare/i })
    fireEvent.click(automatic)
    fireEvent.change(screen.getByLabelText(/rotation interval \(days\)/i), { target: { value: '45' } })
    fireEvent.change(screen.getByLabelText(/staged-key alert after/i), { target: { value: '12' } })
    fireEvent.change(screen.getByLabelText(/grace-expiry alert before/i), { target: { value: '6' } })
    fireEvent.click(screen.getByRole('button', { name: /save policy/i }))

    await waitFor(() => expect(api.updateAgentConfigSigningRotationPolicy).toHaveBeenCalledWith({
      automatic_prepare: true,
      interval_seconds: 3888000,
      staged_alert_after_seconds: 43200,
      grace_alert_before_seconds: 21600,
    }))
  })

  it('renders lifecycle alerts without implying automatic activation', async () => {
    api.fetchAgentConfigSigningKeys.mockResolvedValue(managed([active, staged], {
      health: {
        rotation_due: true,
        staged_pending: true,
        warnings: [
          { code: 'rotation_due', severity: 'warning', kid: active.kid },
          { code: 'staged_pending', severity: 'warning', kid: staged.kid },
        ],
      },
    }))
    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    expect(await screen.findByText(/active signing key is due for rotation/i)).toBeInTheDocument()
    expect(screen.getByText(/auto-staging never activates a key/i)).toBeInTheDocument()
  })

  it('shows external custody health without exposing service credentials', async () => {
    api.fetchAgentConfigSigningKeys.mockResolvedValue({
      provider: 'external-remote', managed: false, grace_period_seconds: 604800, keys: [active],
      provider_info: { name: 'external-remote', external: true, endpoint: 'https://signer.internal', active_kid: active.kid, algorithm: 'Ed25519' },
      provider_health: { healthy: true, checked_at: '2026-08-31T12:00:00Z' },
    })
    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    expect(await screen.findByText(/external key custody/i)).toBeInTheDocument()
    expect(screen.getByText(/signer healthy/i)).toBeInTheDocument()
    expect(screen.getByText('https://signer.internal')).toBeInTheDocument()
    expect(screen.queryByText(/service-secret|bearer/i)).not.toBeInTheDocument()
  })

  it('does not expose lifecycle controls for a legacy provider', async () => {
    api.fetchAgentConfigSigningKeys.mockResolvedValue({ provider: 'local-legacy', managed: false, grace_period_seconds: 604800, keys: [active] })
    render(<I18nProvider><SigningKeysPanel /></I18nProvider>)
    expect(await screen.findByText(/legacy unmanaged signing key/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /prepare rotation/i })).toBeDisabled()
    expect(screen.queryByRole('button', { name: /^activate$/i })).not.toBeInTheDocument()
  })
})
