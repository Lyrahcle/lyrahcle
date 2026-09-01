import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NewAssetForm from './NewAssetForm.jsx'
import * as api from '../lib/api'

vi.mock('../lib/api', () => ({
  createAsset: vi.fn(),
  createProgram: vi.fn(),
  fetchPrograms: vi.fn(),
}))

describe('NewAssetForm', () => {
  beforeEach(() => vi.clearAllMocks())

  it('creates an asset in an existing program', async () => {
    api.fetchPrograms.mockResolvedValue([{ id: 'program-1', name: 'Mission' }])
    api.createAsset.mockResolvedValue({})
    const onCreated = vi.fn()
    render(<NewAssetForm onCreated={onCreated} />)
    fireEvent.click(screen.getByRole('button', { name: /new asset/i }))
    await screen.findByRole('option', { name: 'Mission' })
    fireEvent.change(screen.getByLabelText(/value/i), { target: { value: ' ops.example.com ' } })
    fireEvent.click(screen.getByRole('button', { name: /create/i }))
    await waitFor(() => expect(api.createAsset).toHaveBeenCalledWith({
      programId: 'program-1', kind: 'domain', value: 'ops.example.com',
    }))
    expect(onCreated).toHaveBeenCalledOnce()
  })

  it('requires a program before creating the asset', async () => {
    api.fetchPrograms.mockResolvedValue([])
    render(<NewAssetForm onCreated={() => {}} />)
    fireEvent.click(screen.getByRole('button', { name: /new asset/i }))
    await waitFor(() => expect(api.fetchPrograms).toHaveBeenCalled())
    fireEvent.change(screen.getByLabelText(/value/i), { target: { value: 'example.com' } })
    fireEvent.click(screen.getByRole('button', { name: /create/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/select an existing program/i)
    expect(api.createAsset).not.toHaveBeenCalled()
  })
})
