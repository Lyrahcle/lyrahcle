import LogoMark from './LogoMark.jsx'

const REPO_URL = 'https://github.com/mlko01/lyrahcle'

const LINK_GROUPS = [
  {
    heading: 'Project',
    links: [
      { label: 'GitHub repository', href: REPO_URL },
      { label: 'Security roadmap', href: `${REPO_URL}/blob/main/SECURITY_ROADMAP.md` },
      { label: 'Security policy', href: `${REPO_URL}/blob/main/SECURITY.md` },
    ],
  },
  {
    heading: 'Contact',
    links: [
      { label: 'manu.ciberseguridad@gmail.com', href: 'mailto:manu.ciberseguridad@gmail.com' },
      { label: '@mlko01', href: 'https://github.com/mlko01' },
    ],
  },
  {
    heading: 'Platform',
    lead: 'Open core · AGPLv3',
    links: [
      { label: 'Releasing', href: `${REPO_URL}/blob/main/RELEASING.md` },
      { label: 'Changelog', href: `${REPO_URL}/blob/main/CHANGELOG.md` },
      { label: 'Contributing', href: `${REPO_URL}/blob/main/CONTRIBUTING.md` },
    ],
  },
]

const RELEASE_PHASE = 'Phase 8 · Hardening & v1.0 Prep'

function FooterColumn({ heading, lead, links }) {
  return (
    <div>
      <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-gold">{heading}</p>
      <ul className="mt-3 space-y-2 text-sm text-lyra-muted">
        {lead && (
          <li className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            <span className="h-1.5 w-1.5 rounded-full bg-lyra-good" />
            {lead}
          </li>
        )}
        {links.map(({ label, href }) => (
          <li key={label}>
            <a
              className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
              href={href}
            >
              {label}
            </a>
          </li>
        ))}
      </ul>
    </div>
  )
}

export default function Footer() {
  const year = new Date().getFullYear()

  return (
    <footer className="border-t border-lyra-borderSoft">
      <div className="mx-auto max-w-7xl px-6 py-12 sm:px-8">
        <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <div className="flex items-center gap-2">
              <LogoMark className="h-6 w-6" />
              <span className="font-display text-sm font-medium text-lyra-goldBright">Lyrahcle</span>
            </div>
            <p className="mt-3 max-w-xs text-sm italic text-lyra-mutedDim">
              Never Lies. Mission telemetry for space infrastructure security.
            </p>
          </div>

          {LINK_GROUPS.map((group) => (
            <FooterColumn key={group.heading} {...group} />
          ))}
        </div>

        <div className="mt-10 flex flex-col items-center justify-between gap-3 border-t border-lyra-borderSoft pt-6 sm:flex-row">
          <p className="font-mono text-[11px] text-lyra-mutedDim">
            © {year} Lyrahcle. Core licensed under AGPLv3.
          </p>
          <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-mutedDim">
            {RELEASE_PHASE}
          </p>
        </div>
      </div>
    </footer>
  )
}
