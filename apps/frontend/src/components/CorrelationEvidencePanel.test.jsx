import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import CorrelationEvidencePanel from './CorrelationEvidencePanel.jsx'

const evidence = [
  {
    id: 'summary-1', evidence_type: 'external', evidence_ref: 'cross-domain-v9.4', label: 'Cross-domain correlation summary',
    metadata: {
      confidence: 0.87,
      candidate_severity: 'high',
      corroborating_domains: ['rf', 'detection_rules', 'orbital', 'orbital_operations'],
      context_only_domains: ['asset_alerts'],
      reasons: ['orbital geometry corroborates the RF anomaly', 'asset CVE alerts are context only'],
      severity_applied: false,
      policy: 'explainable_candidate_only',
    },
  },
  {
    id: 'ops-1', evidence_type: 'external', evidence_ref: 'orbital-operations-v9.5', label: 'Maneuver/conjunction context',
    metadata: { candidate_only: true, items: [{ id: 'event-1', event_type: 'maneuver', source: 'Operator', confidence: 0.9, severity: 'medium', event_time: '2026-08-30T18:00:00Z' }] },
  },
  {
    id: 'asset-1', evidence_type: 'external', evidence_ref: 'unacknowledged-7d', label: 'Recent unacknowledged asset CVE alerts',
    metadata: { context_only: true, items: [{ alert_id: 'alert-1', asset_value: 'satcom.example', cve_id: 'CVE-2026-1234', severity: 'high' }] },
  },
]

function renderPanel(items = evidence) {
  return render(<I18nProvider><CorrelationEvidencePanel evidence={items} /></I18nProvider>)
}

describe('CorrelationEvidencePanel', () => {
  it('renders explainable candidate correlation without implying severity was applied', () => {
    renderPanel()
    const panel = screen.getByRole('region', { name: /cross-domain correlation assessment/i })
    expect(within(panel).getByText('87%')).toBeInTheDocument()
    expect(within(panel).getAllByText(/high/i).length).toBeGreaterThanOrEqual(1)
    expect(within(panel).getByText(/not applied to severity/i)).toBeInTheDocument()
    expect(within(panel).getByText('orbital operations')).toBeInTheDocument()
    expect(within(panel).getByText('asset alerts')).toBeInTheDocument()
    expect(within(panel).getByText(/orbital geometry corroborates/i)).toBeInTheDocument()
  })

  it('renders linked orbital and CVE context with candidate/context-only labels', () => {
    renderPanel()
    expect(screen.getByText('Operator')).toBeInTheDocument()
    expect(screen.getByText('CVE-2026-1234')).toBeInTheDocument()
    expect(screen.getByText('satcom.example')).toBeInTheDocument()
    expect(screen.getByText(/candidate only/i)).toBeInTheDocument()
    expect(screen.getByText('asset CVE alerts are context only')).toBeInTheDocument()

    const cveHeading = screen.getByRole('heading', { name: /asset cve context/i })
    const cveCard = cveHeading.parentElement?.parentElement
    expect(cveCard).not.toBeNull()
    expect(within(cveCard).getByText(/^context only$/i)).toBeInTheDocument()
  })

  it('does not render when the incident has no correlation summary evidence', () => {
    const { container } = renderPanel([{ id: 'manual', evidence_ref: 'request-1', metadata: {} }])
    expect(container).toBeEmptyDOMElement()
  })
})
