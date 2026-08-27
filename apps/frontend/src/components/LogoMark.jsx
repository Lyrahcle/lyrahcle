export default function LogoMark({ className = 'h-9 w-9' }) {
  return (
    <svg viewBox="0 0 200 200" className={className} xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="lyraGold" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stopColor="#f0d9a8" />
          <stop offset="100%" stopColor="#b5905e" />
        </linearGradient>
      </defs>
      <circle cx="100" cy="100" r="72" fill="none" stroke="url(#lyraGold)" strokeWidth="3" />
      <g stroke="url(#lyraGold)" strokeWidth="2.5" fill="none" strokeLinecap="round">
        <line x1="72" y1="62" x2="122" y2="52" />
        <line x1="122" y1="52" x2="140" y2="94" />
        <line x1="122" y1="52" x2="98" y2="108" />
        <line x1="72" y1="62" x2="98" y2="108" />
        <line x1="98" y1="108" x2="78" y2="142" />
        <line x1="98" y1="108" x2="118" y2="150" />
      </g>
      <g fill="url(#lyraGold)">
        <circle cx="72" cy="62" r="5" />
        <circle cx="122" cy="52" r="5.5" />
        <circle cx="140" cy="94" r="4.5" />
        <circle cx="98" cy="108" r="6" />
        <circle cx="78" cy="142" r="4.5" />
        <circle cx="118" cy="150" r="5" />
      </g>
    </svg>
  )
}
