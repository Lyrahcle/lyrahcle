import { fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import RateLimitPolicyPanel from './RateLimitPolicyPanel.tsx'
import { I18nProvider } from '../lib/I18nContext'

vi.mock('../lib/api', async (importOriginal) => ({ ...(await importOriginal()), fetchPlatformRateLimits: vi.fn() }))
import { ApiError, fetchPlatformRateLimits } from '../lib/api'

describe('RateLimitPolicyPanel', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows tenant and actor budgets plus distributed state', async () => {
    fetchPlatformRateLimits.mockResolvedValue({
      platform_ceiling_per_minute: 50000,
      tenant_read_per_minute: 1200,
      actor_read_per_minute: 300,
      tenant_write_per_minute: 300,
      actor_write_per_minute: 120,
      tenant_agent_per_minute: 12000,
      agent_per_minute: 3000,
      redis_distributed: true,
      human_and_machine_budgets_separate: true,
    })
    render(<I18nProvider><RateLimitPolicyPanel /></I18nProvider>)
    expect(await screen.findByText(/Redis distributed/i)).toBeInTheDocument()
    expect(screen.getByText('1200')).toBeInTheDocument()
    expect(screen.getByText('3000')).toBeInTheDocument()
    expect(screen.getByText('50000/min')).toBeInTheDocument()
  })

  it('distinguishes an offline backend and retries the request', async () => {
    fetchPlatformRateLimits
      .mockRejectedValueOnce(new ApiError('backend unavailable', { kind: 'network' }))
      .mockResolvedValueOnce({ redis_distributed: true, platform_ceiling_per_minute: 50000 })
    render(<I18nProvider><RateLimitPolicyPanel /></I18nProvider>)
    expect(await screen.findByText(/Service unavailable/i)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Retry/i }))
    expect(await screen.findByText(/Redis distributed/i)).toBeInTheDocument()
  })

  it('surfaces policy load failures', async () => {
    fetchPlatformRateLimits.mockRejectedValue(new Error('policy unavailable'))
    render(<I18nProvider><RateLimitPolicyPanel /></I18nProvider>)
    expect(await screen.findByRole('alert')).toHaveTextContent('policy unavailable')
  })
})
