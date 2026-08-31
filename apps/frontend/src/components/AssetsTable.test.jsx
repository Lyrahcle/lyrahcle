import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext.jsx'
import { clearEffectivePermissions, setEffectivePermissions } from '../lib/permissions.js'
import AssetsTable from './AssetsTable.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({
  updateAsset: vi.fn(),
  deleteAsset: vi.fn(),
}))

const page = {
  items: [{ id: 'asset-1', program_id: 'program-1', kind: 'domain', value: 'ops.example.com' }],
  total: 1,
  limit: 50,
  offset: 0,
}
const programs = [{ id: 'program-1', name: 'Mission Alpha' }, { id: 'program-2', name: 'Mission Beta' }]

function renderTable(props = {}) {
  return render(
    <I18nProvider>
      <AssetsTable page={page} programs={programs} onPrev={() => {}} onNext={() => {}} {...props} />
    </I18nProvider>,
  )
}

describe('AssetsTable management UI', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    clearEffectivePermissions()
  })

  it('shows human program names and hides mutations for read-only users', () => {
    setEffectivePermissions(['assets.read', 'programs.read'])
    renderTable()
    expect(screen.getByText('Mission Alpha')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /delete/i })).not.toBeInTheDocument()
  })

  it('updates an asset from the UI for assets.manage', async () => {
    setEffectivePermissions(['assets.read', 'assets.manage', 'programs.read'])
    api.updateAsset.mockResolvedValue({ id: 'asset-1' })
    const onChanged = vi.fn()
    renderTable({ onChanged })

    fireEvent.click(screen.getByRole('button', { name: /edit/i }))
    fireEvent.change(screen.getByLabelText(/^kind$/i), { target: { value: 'ip' } })
    fireEvent.change(screen.getByLabelText(/^value$/i), { target: { value: ' 198.51.100.42 ' } })
    fireEvent.change(screen.getByLabelText(/^program$/i), { target: { value: 'program-2' } })
    fireEvent.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => expect(api.updateAsset).toHaveBeenCalledWith('asset-1', {
      programId: 'program-2',
      kind: 'ip',
      value: '198.51.100.42',
    }))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('deletes an asset only after confirmation', async () => {
    setEffectivePermissions(['assets.read', 'assets.manage'])
    api.deleteAsset.mockResolvedValue(undefined)
    const onChanged = vi.fn()
    renderTable({ onChanged })

    fireEvent.click(screen.getByRole('button', { name: /^delete$/i }))
    expect(api.deleteAsset).not.toHaveBeenCalled()

    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^delete$/i }))

    await waitFor(() => expect(api.deleteAsset).toHaveBeenCalledWith('asset-1'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('does not delete an asset when the confirmation is cancelled', async () => {
    setEffectivePermissions(['assets.read', 'assets.manage'])
    renderTable()

    fireEvent.click(screen.getByRole('button', { name: /^delete$/i }))
    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^cancel$/i }))

    expect(api.deleteAsset).not.toHaveBeenCalled()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })
})
