import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import ConfirmDialog from './ConfirmDialog.jsx'
import { I18nProvider } from '../lib/I18nContext.jsx'

function renderDialog(props = {}) {
  const onConfirm = vi.fn()
  const onCancel = vi.fn()
  render(
    <I18nProvider>
      <ConfirmDialog message="Delete this item?" onConfirm={onConfirm} onCancel={onCancel} {...props} />
    </I18nProvider>,
  )
  return { onConfirm, onCancel }
}

describe('ConfirmDialog', () => {
  it('renders the message and calls onConfirm/onCancel', () => {
    const { onConfirm, onCancel } = renderDialog()
    expect(screen.getByText('Delete this item?')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /confirm/i }))
    expect(onConfirm).toHaveBeenCalledOnce()

    fireEvent.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onCancel).toHaveBeenCalledOnce()
  })

  it('uses custom labels and title when provided', () => {
    renderDialog({ title: 'Delete asset', confirmLabel: 'Delete', cancelLabel: 'Keep it' })
    expect(screen.getByText('Delete asset')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Keep it' })).toBeInTheDocument()
  })

  it('calls onCancel when Escape is pressed', () => {
    const { onCancel } = renderDialog()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onCancel).toHaveBeenCalledOnce()
  })

  it('disables both buttons and ignores Escape while busy', () => {
    const { onCancel } = renderDialog({ busy: true })
    expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /saving/i })).toBeDisabled()

    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onCancel).not.toHaveBeenCalled()
  })

  it('exposes the dialog with alertdialog semantics for screen readers', () => {
    renderDialog()
    const dialog = screen.getByRole('alertdialog')
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAttribute('aria-labelledby', 'confirm-dialog-title')
    expect(dialog).toHaveAttribute('aria-describedby', 'confirm-dialog-message')
  })
})
