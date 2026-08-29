const NODES = [
  { id: 'a', x: 200, y: 70, r: 5, delay: '0s' },
  { id: 'b', x: 300, y: 120, r: 6, delay: '0.4s' },
  { id: 'c', x: 330, y: 210, r: 4.5, delay: '0.8s' },
  { id: 'd', x: 250, y: 260, r: 6.5, delay: '1.2s' },
  { id: 'e', x: 150, y: 260, r: 4.5, delay: '1.6s' },
  { id: 'f', x: 90, y: 190, r: 5.5, delay: '2s' },
  { id: 'g', x: 120, y: 110, r: 4, delay: '2.4s' },
  { id: 'h', x: 200, y: 170, r: 7, delay: '0.2s' },
]

const LINKS = [
  ['a', 'b'],
  ['b', 'c'],
  ['c', 'd'],
  ['d', 'e'],
  ['e', 'f'],
  ['f', 'g'],
  ['g', 'a'],
  ['h', 'a'],
  ['h', 'c'],
  ['h', 'e'],
  ['h', 'g'],
]

function node(id) {
  return NODES.find((n) => n.id === id)
}

export default function LoginConstellation() {
  return (
    <div className="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden">
      <div className="motion-safe:animate-[spin_140s_linear_infinite] absolute h-[480px] w-[480px] rounded-full border border-dashed border-lyra-gold/[0.08]" />
      <div className="motion-safe:animate-[spin_100s_linear_infinite_reverse] absolute h-[360px] w-[360px] rounded-full border border-lyra-gold/[0.06]" />

      <svg
        viewBox="0 0 400 330"
        className="motion-safe:animate-[spin_220s_linear_infinite] h-[420px] w-[420px] opacity-60"
        xmlns="http://www.w3.org/2000/svg"
        aria-hidden="true"
        focusable="false"
      >
        <g stroke="rgb(var(--lyra-gold))" strokeOpacity="0.18" strokeWidth="1.2" fill="none">
          {LINKS.map(([from, to]) => {
            const a = node(from)
            const b = node(to)
            return <line key={`${from}-${to}`} x1={a.x} y1={a.y} x2={b.x} y2={b.y} />
          })}
        </g>
        <g fill="rgb(var(--lyra-gold-bright))">
          {NODES.map((n) => (
            <circle
              key={n.id}
              cx={n.x}
              cy={n.y}
              r={n.r}
              className="lyra-pulse-node"
              style={{ animationDelay: n.delay }}
            />
          ))}
        </g>
      </svg>
    </div>
  )
}
