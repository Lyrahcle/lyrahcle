import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import Pagination from './Pagination.jsx'
import { I18nProvider } from '../lib/I18nContext'

function renderPagination(props) {
  return render(<I18nProvider><Pagination {...props} /></I18nProvider>)
}

describe('Pagination', () => {
  it('stays hidden when all rows fit on one page', () => {
    const { container } = renderPagination({ total: 10, limit: 10, offset: 0, onPrev: () => {}, onNext: () => {} })
    expect(container).toBeEmptyDOMElement()
  })

  it('disables previous on the first page and advances', () => {
    const onNext = vi.fn()
    renderPagination({ total: 25, limit: 10, offset: 0, onPrev: () => {}, onNext })
    expect(screen.getByRole('button', { name: /prev/i })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: /next/i }))
    expect(onNext).toHaveBeenCalledOnce()
    expect(screen.getByText(/1–10/)).toBeInTheDocument()
  })

  it('disables next on the last page', () => {
    renderPagination({ total: 25, limit: 10, offset: 20, onPrev: () => {}, onNext: () => {} })
    expect(screen.getByRole('button', { name: /next/i })).toBeDisabled()
    expect(screen.getByText(/21–25/)).toBeInTheDocument()
  })
})
