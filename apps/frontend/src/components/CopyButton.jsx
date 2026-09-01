import { useEffect, useRef, useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { useI18n } from '../lib/I18nContext.jsx'

const RESET_DELAY_MS = 1500

export default function CopyButton({ value, label, className = '' }) {
  const { t } = useI18n()
  const [copied, setCopied] = useState(false)
  const timeoutRef = useRef(null)

  useEffect(() => () => window.clearTimeout(timeoutRef.current), [])

  async function handleCopy(event) {
    event.stopPropagation()
    if (value == null || value === '') return
    try {
      await navigator.clipboard.writeText(String(value))
      setCopied(true)
      timeoutRef.current = window.setTimeout(() => setCopied(false), RESET_DELAY_MS)
    } catch {
      setCopied(false)
    }
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      aria-label={copied ? t('common.copied') : t('common.copyValue', { label })}
      title={copied ? t('common.copied') : t('common.copyValue', { label })}
      className={`inline-flex shrink-0 items-center justify-center rounded p-1 text-lyra-mutedDim outline-none transition-colors hover:bg-lyra-panel hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold ${className}`}
    >
      {copied ? (
        <Check className="h-3.5 w-3.5 text-lyra-good" aria-hidden="true" />
      ) : (
        <Copy className="h-3.5 w-3.5" aria-hidden="true" />
      )}
    </button>
  )
}
