export default function Panel({ eyebrow, title, children, bleed = false, uppercase = false, titleExtra }) {
  return (
    <section className="overflow-hidden rounded-lg border border-lyra-border bg-lyra-panel/30">
      <div className="px-5 pb-5 pt-5 sm:px-7 sm:pt-7">
        <div className="mb-4 flex items-baseline gap-3">
          <span className="font-mono text-xs uppercase tracking-[0.2em] text-lyra-gold">
            {eyebrow}
          </span>
          <div className="h-px flex-1 bg-lyra-borderSoft" />
        </div>
        <div className="flex items-center gap-3">
          <h2 className={`font-display text-lg font-medium text-lyra-text ${uppercase ? 'uppercase tracking-wide' : ''}`}>
            {title}
          </h2>
          {titleExtra}
        </div>
      </div>
      {bleed ? children : <div className="px-5 pb-5 sm:px-7 sm:pb-7">{children}</div>}
    </section>
  )
}
