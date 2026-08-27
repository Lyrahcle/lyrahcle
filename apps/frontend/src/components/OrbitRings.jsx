export default function OrbitRings() {
  return (
    <div className="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden">
      <div className="motion-safe:animate-[spin_60s_linear_infinite] h-[420px] w-[420px] rounded-full border border-dashed border-lyra-gold/10" />
      <div className="motion-safe:animate-[spin_90s_linear_infinite_reverse] absolute h-[320px] w-[320px] rounded-full border border-lyra-gold/10" />
      <div className="absolute h-[220px] w-[220px] rounded-full border border-lyra-gold/[0.08]" />
    </div>
  )
}
