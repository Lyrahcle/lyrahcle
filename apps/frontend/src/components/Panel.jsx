export default function Panel({ eyebrow, title, children }) {
  return (
    <section className="rounded-lg border border-lyra-border bg-lyra-panel/30 p-4 sm:p-6">
      <div className="mb-5 flex items-baseline gap-3">
        <span className="font-mono text-[10px] uppercase tracking-[0.2em] text-lyra-gold">
          {eyebrow}
        </span>
        <div className="h-px flex-1 bg-lyra-borderSoft" />
      </div>
      <h2 className="font-display -mt-3 mb-4 text-base font-medium text-lyra-text">{title}</h2>
      {children}
    </section>
  )
}
