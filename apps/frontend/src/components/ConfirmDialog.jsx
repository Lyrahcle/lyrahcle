import { createPortal } from 'react-dom'
import { useI18n } from '../lib/I18nContext'
import { useFocusTrap } from '../lib/useFocusTrap'

export default function ConfirmDialog({
  title,
  message,
  confirmLabel,
  cancelLabel,
  danger = true,
  busy = false,
  onConfirm,
  onCancel,
}) {
  const { t } = useI18n()
  const dialogRef = useFocusTrap(true, busy ? undefined : onCancel)

  return createPortal(
    <div className="fixed inset-0 z-[100] overflow-y-auto bg-black/70 p-4 backdrop-blur-[2px] sm:p-6">
      <div className="flex min-h-full items-center justify-center">
        <div
          ref={dialogRef}
          role="alertdialog"
          aria-modal="true"
          aria-labelledby="confirm-dialog-title"
          aria-describedby="confirm-dialog-message"
          tabIndex={-1}
          className="w-full max-w-md overflow-hidden rounded-xl border border-lyra-border bg-lyra-panel p-5 shadow-2xl shadow-black/40 outline-none sm:p-6"
        >
          <h2 id="confirm-dialog-title" className="font-display text-sm font-medium text-lyra-goldBright">
            {title || t('common.confirmActionTitle')}
          </h2>
          <p id="confirm-dialog-message" className="mt-3 text-sm text-lyra-muted">
            {message}
          </p>
          <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
            <button
              type="button"
              disabled={busy}
              onClick={onCancel}
              className="rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold disabled:opacity-50"
            >
              {cancelLabel || t('common.cancel')}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={onConfirm}
              className={`rounded px-4 py-2 text-sm font-medium outline-none transition-opacity hover:opacity-90 disabled:opacity-50 ${
                danger
                  ? 'border border-lyra-bad text-lyra-bad focus-visible:ring-2 focus-visible:ring-lyra-bad'
                  : 'bg-lyra-gold text-lyra-bg focus-visible:ring-2 focus-visible:ring-lyra-goldBright'
              }`}
            >
              {busy ? t('common.saving') : confirmLabel || t('common.confirm')}
            </button>
          </div>
        </div>
      </div>
    </div>,
    document.body,
  )
}
