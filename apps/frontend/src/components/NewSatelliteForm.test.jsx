import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NewSatelliteForm from './NewSatelliteForm.jsx'
import * as api from '../lib/api.js'

vi.mock('../lib/api.js', () => ({ createSatellite: vi.fn() }))

describe('NewSatelliteForm', () => {
  beforeEach(() => vi.clearAllMocks())

  it('submits trimmed TLE data and numeric NORAD id', async () => {
    api.createSatellite.mockResolvedValue({})
    const onCreated = vi.fn()
    render(<NewSatelliteForm onCreated={onCreated} />)
    fireEvent.click(screen.getByRole('button', { name: /register satellite/i }))
    fireEvent.change(screen.getByLabelText(/NORAD ID/i), { target: { value: '25544' } })
    fireEvent.change(screen.getByLabelText(/^Name$/i), { target: { value: ' ISS ' } })
    fireEvent.change(screen.getByLabelText(/TLE line 1/i), { target: { value: ' 1 TEST ' } })
    fireEvent.change(screen.getByLabelText(/TLE line 2/i), { target: { value: ' 2 TEST ' } })
    fireEvent.click(screen.getByRole('button', { name: /^register$/i }))
    await waitFor(() => expect(api.createSatellite).toHaveBeenCalledWith({
      noradId: 25544, name: 'ISS', tleLine1: '1 TEST', tleLine2: '2 TEST',
    }))
    expect(onCreated).toHaveBeenCalledOnce()
  })

  it('shows API failures without calling onCreated', async () => {
    api.createSatellite.mockRejectedValue(new Error('satellite rejected'))
    const onCreated = vi.fn()
    render(<NewSatelliteForm onCreated={onCreated} />)
    fireEvent.click(screen.getByRole('button', { name: /register satellite/i }))
    fireEvent.change(screen.getByLabelText(/NORAD ID/i), { target: { value: '25544' } })
    fireEvent.change(screen.getByLabelText(/^Name$/i), { target: { value: 'ISS' } })
    fireEvent.change(screen.getByLabelText(/TLE line 1/i), { target: { value: '1 TEST' } })
    fireEvent.change(screen.getByLabelText(/TLE line 2/i), { target: { value: '2 TEST' } })
    fireEvent.click(screen.getByRole('button', { name: /^register$/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('satellite rejected')
    expect(onCreated).not.toHaveBeenCalled()
  })
})
