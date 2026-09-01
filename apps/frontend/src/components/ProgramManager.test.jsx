import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import { clearEffectivePermissions, setEffectivePermissions } from '../lib/permissions'
import ProgramManager from './ProgramManager.jsx'
import * as api from '../lib/api'

vi.mock('../lib/api', () => ({
  updateProgram: vi.fn(),
  deleteProgram: vi.fn(),
}))

const programs = [{ id: 'program-1', name: 'Mission Alpha' }]

function renderManager(props = {}) {
  return render(<I18nProvider><ProgramManager programs={programs} {...props} /></I18nProvider>)
}

describe('ProgramManager', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    clearEffectivePermissions()
  })

  it('is read-only without programs.manage', () => {
    setEffectivePermissions(['programs.read'])
    renderManager()
    expect(screen.getByText('Mission Alpha')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /edit/i })).not.toBeInTheDocument()
  })

  it('renames programs from the UI', async () => {
    setEffectivePermissions(['programs.read', 'programs.manage'])
    api.updateProgram.mockResolvedValue({ id: 'program-1', name: 'Mission Beta' })
    const onChanged = vi.fn()
    renderManager({ onChanged })

    fireEvent.click(screen.getByRole('button', { name: /edit/i }))
    fireEvent.change(screen.getByLabelText(/program name/i), { target: { value: ' Mission Beta ' } })
    fireEvent.click(screen.getByRole('button', { name: /^save$/i }))

    await waitFor(() => expect(api.updateProgram).toHaveBeenCalledWith('program-1', 'Mission Beta'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('deletes a program after a destructive confirmation', async () => {
    setEffectivePermissions(['programs.read', 'programs.manage'])
    api.deleteProgram.mockResolvedValue(undefined)
    const onChanged = vi.fn()
    renderManager({ onChanged })

    fireEvent.click(screen.getByRole('button', { name: /^delete$/i }))
    expect(api.deleteProgram).not.toHaveBeenCalled()

    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^delete$/i }))

    await waitFor(() => expect(api.deleteProgram).toHaveBeenCalledWith('program-1'))
    expect(onChanged).toHaveBeenCalledOnce()
  })

  it('does not delete a program when the confirmation is cancelled', async () => {
    setEffectivePermissions(['programs.read', 'programs.manage'])
    renderManager()

    fireEvent.click(screen.getByRole('button', { name: /^delete$/i }))
    const dialog = await screen.findByRole('alertdialog')
    fireEvent.click(within(dialog).getByRole('button', { name: /^cancel$/i }))

    expect(api.deleteProgram).not.toHaveBeenCalled()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  })
})
