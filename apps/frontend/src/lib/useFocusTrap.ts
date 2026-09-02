import { useEffect, useRef, type RefObject } from 'react'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'textarea:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(', ')

function isAvailableForFocus(element: HTMLElement): boolean {
  if (element.hasAttribute('hidden')) return false
  if (element.getAttribute('aria-hidden') === 'true') return false
  if (element.closest('[inert]')) return false
  return true
}

export function useFocusTrap(
  active: boolean = true,
  onEscape?: () => void,
): RefObject<HTMLElement | null> {
  const containerRef = useRef<HTMLElement | null>(null)
  const onEscapeRef = useRef(onEscape)

  useEffect(() => {
    onEscapeRef.current = onEscape
  }, [onEscape])

  useEffect(() => {
    if (!active) return
    const container = containerRef.current
    if (!container) return

    const previouslyFocused = document.activeElement
    const focusables = (): HTMLElement[] =>
      (Array.from(container.querySelectorAll(FOCUSABLE_SELECTOR)) as HTMLElement[]).filter(isAvailableForFocus)

    const first = focusables()[0]
    ;(first ?? container).focus()

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape' && onEscapeRef.current) {
        event.preventDefault()
        event.stopPropagation()
        onEscapeRef.current()
        return
      }

      if (event.key !== 'Tab') return
      const elements = focusables()
      if (elements.length === 0) {
        event.preventDefault()
        container?.focus()
        return
      }

      const firstElement = elements[0]
      const lastElement = elements[elements.length - 1]

      if (event.shiftKey && (document.activeElement === firstElement || document.activeElement === container)) {
        event.preventDefault()
        lastElement.focus()
      } else if (!event.shiftKey && document.activeElement === lastElement) {
        event.preventDefault()
        firstElement.focus()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      if (previouslyFocused instanceof HTMLElement && previouslyFocused.isConnected) {
        previouslyFocused.focus()
      }
    }
  }, [active])

  return containerRef
}
