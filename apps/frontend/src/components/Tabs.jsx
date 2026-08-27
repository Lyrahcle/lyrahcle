export default function Tabs({ tabs, active, onChange }) {
  return (
    <div className="mb-6 flex gap-1 border-b border-lyra-borderSoft">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          onClick={() => onChange(tab.id)}
          className={`relative px-4 py-2.5 font-mono text-[11px] uppercase tracking-wider transition-colors ${
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
