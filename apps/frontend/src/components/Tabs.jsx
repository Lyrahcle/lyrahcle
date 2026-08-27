export default function Tabs({ tabs, active, onChange }) {
  return (
    <div className="mb-7 flex gap-1 border-b border-lyra-borderSoft">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          onClick={() => onChange(tab.id)}
          className={`relative rounded-sm px-5 py-3 font-mono text-xs uppercase tracking-wider outline-none transition-colors focus-visible:ring-1 focus-visible:ring-lyra-gold ${
            active === tab.id
              ? 'text-lyra-goldBright'
              : 'text-lyra-mutedDim hover:text-lyra-muted'
          }`}
        >
          {tab.label}
          {active === tab.id && (
            <span className="absolute inset-x-0 -bottom-px h-px bg-lyra-gold" />
          )}
        </button>
      ))}
    </div>
  )
}
