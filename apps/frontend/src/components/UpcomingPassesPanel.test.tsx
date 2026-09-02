import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import UpcomingPassesPanel from './UpcomingPassesPanel'

const pass = { aos: '2026-09-02T10:00:00Z', peak_at: '2026-09-02T10:05:00Z', los: '2026-09-02T10:10:00Z', max_elevation_deg: 42.25, duration_seconds: 600 }
const now = Date.parse('2026-09-02T09:30:00Z')

function view(props: Record<string, unknown> = {}) {
  return render(<I18nProvider><UpcomingPassesPanel passes={[pass]} stationId="gs-madrid" agentStatus="active" agentLastSeenAt="2026-09-02T09:29:00Z" now={now} access={{ rf: true, incidents: true }} {...props} /></I18nProvider>)
}

describe('UpcomingPassesPanel', () => {
  it('shows real pass geometry and a nominal operational state', () => {
    view()
    expect(screen.getByRole('heading', { name: /upcoming passes/i })).toBeInTheDocument()
    expect(screen.getByText('42.3°')).toBeInTheDocument()
    expect(screen.getByText(/nominal/i)).toBeInTheDocument()
    expect(screen.getByText(/gs-madrid/i)).toBeInTheDocument()
  })

  it('prioritizes an active incident over nominal pass state', () => {
    view({ activeIncidentCount: 1 })
    expect(screen.getByText(/incident active/i)).toBeInTheDocument()
  })

  it('marks an unavailable ground agent without inventing telemetry', () => {
    view({ agentStatus: 'revoked' })
    expect(screen.getByText(/agent offline/i)).toBeInTheDocument()
  })

  it('opens the existing RF workspace', () => {
    const onNavigate = vi.fn()
    view({ onNavigate })
    fireEvent.click(screen.getByRole('button', { name: /open rf workspace/i }))
    expect(onNavigate).toHaveBeenCalledWith('signal')
  })
})
