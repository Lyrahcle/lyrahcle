import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { I18nProvider } from '../lib/I18nContext'
import { ToastProvider, useToast } from './ToastProvider'

function Trigger() {
  const { notify } = useToast()
  return (
    <div>
      <button type="button" onClick={() => notify({ title: 'Saved', tone: 'success', dedupeKey: 'saved' })}>Success</button>
      <button type="button" onClick={() => notify({ title: 'Failed', message: 'Backend unavailable', tone: 'error' })}>Error</button>
    </div>
  )
}

function renderToastSystem() {
  return render(<I18nProvider><ToastProvider><Trigger /></ToastProvider></I18nProvider>)
}

describe('ToastProvider', () => {
  it('announces routine feedback politely and deduplicates repeated events', () => {
    renderToastSystem()
    fireEvent.click(screen.getByRole('button', { name: 'Success' }))
    fireEvent.click(screen.getByRole('button', { name: 'Success' }))
    expect(screen.getAllByRole('status')).toHaveLength(1)
    expect(screen.getByRole('status')).toHaveTextContent('Saved')
  })

  it('keeps error feedback assertive and dismissible', () => {
    renderToastSystem()
    fireEvent.click(screen.getByRole('button', { name: 'Error' }))
    expect(screen.getByRole('alert')).toHaveTextContent('Backend unavailable')
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })
})
