import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TMTCSequenceAnalyzer from './TMTCSequenceAnalyzer'
import { analyzeCCSDSSequence } from '../lib/api'

vi.mock('../lib/api', () => ({ analyzeCCSDSSequence: vi.fn() }))

const analyzeMock = vi.mocked(analyzeCCSDSSequence)

describe('TMTCSequenceAnalyzer', () => {
  beforeEach(() => analyzeMock.mockReset())

  it('sends only non-empty captured packet lines in chronological order', async () => {
    analyzeMock.mockResolvedValue({ samples: [], issues: [], replay_candidates: 0, sequence_gaps: 0, out_of_order: 0, collisions: 0 })
    render(<TMTCSequenceAnalyzer />)
    fireEvent.change(screen.getByLabelText('Captured CCSDS packets'), { target: { value: 'aa\n\n bb \n' } })
    fireEvent.click(screen.getByRole('button', { name: 'Analyze sequence' }))
    await waitFor(() => expect(analyzeMock).toHaveBeenCalledWith(['aa', 'bb']))
    expect(await screen.findByText('No sequence anomalies detected in this capture.')).toBeInTheDocument()
  })

  it('shows replay candidates as evidence requiring correlation', async () => {
    analyzeMock.mockResolvedValue({ samples: [{}, {}], replay_candidates: 1, sequence_gaps: 0, out_of_order: 1, collisions: 0, issues: [{ index: 1, apid: 42, packet_type: 'TC', sequence_count: 10, kind: 'replay_candidate', detail: 'exact packet content repeats sequence count first observed at index 0', replay_candidate: true }] })
    render(<TMTCSequenceAnalyzer />)
    fireEvent.change(screen.getByLabelText('Captured CCSDS packets'), { target: { value: 'aa\naa' } })
    fireEvent.click(screen.getByRole('button', { name: 'Analyze sequence' }))
    expect(await screen.findByText('Replay candidate')).toBeInTheDocument()
    expect(screen.getByText(/cannot establish malicious intent/i)).toBeInTheDocument()
  })
})
