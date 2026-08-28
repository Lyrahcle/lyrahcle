import { Search } from 'lucide-react'
import { useState } from 'react'

export default function SearchBar() {
  const [value, setValue] = useState('')

  return (
    <label className="group hidden items-center gap-2 rounded border border-lyra-border bg-lyra-panel/40 px-3 py-1.5 transition-colors focus-within:border-lyra-gold/50 md:flex">
      <Search className="h-3.5 w-3.5 text-lyra-mutedDim group-focus-within:text-lyra-gold" strokeWidth={1.75} />
      <input
        value={value}
        onChange={(e) => setValue(e.target.value)}
        type="text"
        placeholder="Search assets, satellites, CVEs…"
        className="w-56 bg-transparent font-mono text-[11px] text-lyra-text placeholder:text-lyra-mutedDim focus:outline-none"
      />
      <kbd className="rounded border border-lyra-borderSoft px-1.5 py-0.5 font-mono text-[10px] text-lyra-mutedDim">
        /
      </kbd>
    </label>
  )
}
