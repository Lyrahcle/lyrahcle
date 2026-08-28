import LogoMark from './LogoMark.jsx'

export default function Footer() {
  const year = new Date().getFullYear()

  return (
    <footer className="border-t border-lyra-borderSoft">
      <div className="mx-auto max-w-7xl px-6 py-12 sm:px-8">
        <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <div className="flex items-center gap-2">
              <LogoMark className="h-6 w-6" />
              <span className="font-display text-sm font-medium text-lyra-goldBright">
                Lyrahcle
              </span>
            </div>
            <p className="mt-3 max-w-xs text-sm italic text-lyra-mutedDim">
              Never Lies. Mission telemetry for space infrastructure security.
            </p>
          </div>

          <div>
            <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-gold">
              Project
            </p>
            <ul className="mt-3 space-y-2 text-sm text-lyra-muted">
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01/lyrahcle"
                >
                  GitHub repository
                </a>
              </li>
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01/lyrahcle/blob/main/PLAN.md"
                >
                  Development plan
                </a>
              </li>
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01/lyrahcle/blob/main/SECURITY_ROADMAP.md"
                >
                  Security roadmap
                </a>
              </li>
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01/lyrahcle/blob/main/SECURITY.md"
                >
                  Security policy
                </a>
              </li>
            </ul>
          </div>

          <div>
            <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-gold">
              Contact
            </p>
            <ul className="mt-3 space-y-2 text-sm text-lyra-muted">
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="mailto:manu.ciberseguridad@gmail.com"
                >
                  manu.ciberseguridad@gmail.com
                </a>
              </li>
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01"
                >
                  @mlko01
                </a>
              </li>
            </ul>
          </div>

          <div>
            <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-gold">
              Platform
            </p>
            <ul className="mt-3 space-y-2 text-sm text-lyra-muted">
              <li className="flex items-center gap-2 font-mono text-[11px] uppercase tracking-wider text-lyra-mutedDim">
                <span className="h-1.5 w-1.5 rounded-full bg-lyra-good" />
                Open core · AGPLv3
              </li>
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01/lyrahcle/blob/main/RELEASING.md"
                >
                  Releasing
                </a>
              </li>
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01/lyrahcle/blob/main/CHANGELOG.md"
                >
                  Changelog
                </a>
              </li>
              <li>
                <a
                  className="outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
                  href="https://github.com/mlko01/lyrahcle/blob/main/CONTRIBUTING.md"
                >
                  Contributing
                </a>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-10 flex flex-col items-center justify-between gap-3 border-t border-lyra-borderSoft pt-6 sm:flex-row">
          <p className="font-mono text-[11px] text-lyra-mutedDim">
            © {year} Lyrahcle. Core licensed under AGPLv3.
          </p>
          <p className="font-mono text-[11px] uppercase tracking-[0.2em] text-lyra-mutedDim">
            Phase 8 · Hardening &amp; v1.0 Prep
          </p>
        </div>
      </div>
    </footer>
  )
}
