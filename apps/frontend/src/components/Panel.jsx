export default function Panel({ eyebrow, title, children }) {
  return (
    <section className="rounded-lg border border-lyra-border bg-lyra-panel/30 p-5 sm:p-7">
      <div className="mb-6 flex items-baseline gap-3">
        <span className="font-mono text-xs uppercase tracking-[0.2em] text-lyra-gold">
          {eyebrow}
        </span>
        <div className="h-px flex-1 bg-lyra-borderSoft" />
      </div>
      <h2 className="font-display -mt-4 mb-5 text-lg font-medium text-lyra-text">{title}</h2>
      {children}
    </section>
  )
}
