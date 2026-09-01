import { useI18n } from '../lib/I18nContext.jsx'

export default function Pagination({ total, limit, offset, onPrev, onNext }) {
  const { t } = useI18n()
  if (total <= limit) return null

  const start = total === 0 ? 0 : offset + 1
  const end = Math.min(offset + limit, total)

  return (
    <div className="mt-4 flex items-center justify-between border-t border-lyra-borderSoft pt-3">
      <p className="font-mono text-[11px] tracking-wider text-lyra-mutedDim">
        {start}–{end} {t('common.of')} {total}
      </p>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={onPrev}
          disabled={offset === 0}
          className="rounded border border-lyra-border px-3 py-1 font-mono text-[11px] tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold disabled:cursor-not-allowed disabled:opacity-40"
        >
          {t('common.prev')}
        </button>
        <button
          type="button"
          onClick={onNext}
          disabled={offset + limit >= total}
          className="rounded border border-lyra-border px-3 py-1 font-mono text-[11px] tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold disabled:cursor-not-allowed disabled:opacity-40"
        >
          {t('common.next')}
        </button>
      </div>
    </div>
  )
}
