import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { AlertTriangle, CheckCircle2, CircleX, Info, X } from 'lucide-react'
import { useI18n } from '../lib/I18nContext'

export type ToastTone = 'success' | 'info' | 'warning' | 'error'

export interface ToastAction {
  label: string
  onClick: () => void
}

export interface ToastInput {
  title: string
  message?: string
  tone?: ToastTone
  durationMs?: number
  dedupeKey?: string
  action?: ToastAction
}

interface ToastRecord extends ToastInput {
  id: string
  tone: ToastTone
  durationMs: number
}

interface ToastContextValue {
  notify: (toast: ToastInput) => string
  dismiss: (id: string) => void
  dismissAll: () => void
}

const MAX_TOASTS = 4
let sequence = 0

const fallbackContext: ToastContextValue = {
  notify: () => '',
  dismiss: () => {},
  dismissAll: () => {},
}

const ToastContext = createContext<ToastContextValue>(fallbackContext)

function defaultDuration(tone: ToastTone): number {
  return tone === 'error' || tone === 'warning' ? 0 : 6000
}

function toneIcon(tone: ToastTone) {
  if (tone === 'success') return CheckCircle2
  if (tone === 'warning') return AlertTriangle
  if (tone === 'error') return CircleX
  return Info
}

function toneClass(tone: ToastTone): string {
  if (tone === 'success') return 'border-lyra-good/40 text-lyra-good'
  if (tone === 'warning') return 'border-lyra-warn/40 text-lyra-warn'
  if (tone === 'error') return 'border-lyra-bad/40 text-lyra-bad'
  return 'border-lyra-gold/40 text-lyra-goldBright'
}

function ToastItem({ toast, onDismiss }: { toast: ToastRecord; onDismiss: (id: string) => void }) {
  const { t } = useI18n()
  const Icon = toneIcon(toast.tone)
  const urgent = toast.tone === 'error' || toast.tone === 'warning'

  useEffect(() => {
    if (toast.durationMs <= 0) return undefined
    const timer = window.setTimeout(() => onDismiss(toast.id), toast.durationMs)
    return () => window.clearTimeout(timer)
  }, [toast.id, toast.durationMs, onDismiss])

  return (
    <section
      role={urgent ? 'alert' : 'status'}
      aria-live={urgent ? 'assertive' : 'polite'}
      aria-atomic="true"
      className={`pointer-events-auto rounded-lg border bg-lyra-panel/95 p-4 shadow-2xl backdrop-blur ${toneClass(toast.tone)}`}
    >
      <div className="flex items-start gap-3">
        <Icon aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-lyra-text">{toast.title}</p>
          {toast.message && <p className="mt-1 text-xs leading-5 text-lyra-muted">{toast.message}</p>}
          {toast.action && (
            <button
              type="button"
              onClick={() => { toast.action?.onClick(); onDismiss(toast.id) }}
              className="mt-3 rounded border border-current/40 px-3 py-1.5 font-mono text-[11px] tracking-wider outline-none focus-visible:ring-2 focus-visible:ring-lyra-goldBright/70"
            >
              {toast.action.label}
            </button>
          )}
        </div>
        <button
          type="button"
          onClick={() => onDismiss(toast.id)}
          aria-label={t('common.close')}
          className="rounded p-1 text-lyra-muted outline-none hover:text-lyra-text focus-visible:ring-2 focus-visible:ring-lyra-goldBright/70"
        >
          <X aria-hidden="true" className="h-4 w-4" />
        </button>
      </div>
    </section>
  )
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const { t } = useI18n()
  const [toasts, setToasts] = useState<ToastRecord[]>([])

  const dismiss = useCallback((id: string) => {
    setToasts((current) => current.filter((toast) => toast.id !== id))
  }, [])

  const dismissAll = useCallback(() => setToasts([]), [])

  const notify = useCallback((input: ToastInput) => {
    const tone = input.tone ?? 'info'
    const id = `toast-${Date.now().toString(36)}-${sequence += 1}`
    const next: ToastRecord = {
      ...input,
      id,
      tone,
      durationMs: input.durationMs ?? defaultDuration(tone),
    }
    setToasts((current) => {
      const withoutDuplicate = input.dedupeKey ? current.filter((toast) => toast.dedupeKey !== input.dedupeKey) : current
      return [...withoutDuplicate, next].slice(-MAX_TOASTS)
    })
    return id
  }, [])

  const value = useMemo<ToastContextValue>(() => ({ notify, dismiss, dismissAll }), [notify, dismiss, dismissAll])

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div
        className="pointer-events-none fixed right-4 top-4 z-[120] grid w-[min(24rem,calc(100vw-2rem))] gap-3 sm:right-6 sm:top-6"
        aria-label={t('ux.notifications.region')}
      >
        {toasts.map((toast) => <ToastItem key={toast.id} toast={toast} onDismiss={dismiss} />)}
      </div>
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  return useContext(ToastContext)
}
