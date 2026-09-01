import LogoMark from './LogoMark.jsx'
import { useI18n } from '../lib/I18nContext'
import pkg from '../../package.json'

const REPO_URL = 'https://github.com/mlko01/lyrahcle'

function FooterColumn({ heading, lead, links }) {
  return (
    <div>
      <p className="font-mono text-[11px] tracking-[0.2em] text-lyra-gold">{heading}</p>
      <ul className="mt-3 space-y-2 text-sm text-lyra-muted">
        {lead && (
          <li className="flex items-center gap-2 font-mono text-[11px] tracking-wider text-lyra-mutedDim">
            <span className="h-1.5 w-1.5 rounded-full bg-lyra-good" />
            {lead}
          </li>
        )}
        {links.map(({ label, href }) => (
          <li key={label}>
            <a
              className="font-mono text-[11px] tracking-wider outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
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
  const { t } = useI18n()
  const year = new Date().getFullYear()

  const linkGroups = [
    {
      heading: t('footer.project'),
      links: [
        { label: t('footer.githubRepo'), href: REPO_URL },
        { label: t('footer.securityRoadmap'), href: `${REPO_URL}/blob/main/SECURITY_ROADMAP.md` },
        { label: t('footer.securityPolicy'), href: `${REPO_URL}/blob/main/SECURITY.md` },
        { label: t('footer.documentation'), href: `${REPO_URL}/tree/main/docs` },
      ],
    },
    {
      heading: t('footer.contact'),
      links: [
        { label: 'manu.ciberseguridad@gmail.com', href: 'mailto:manu.ciberseguridad@gmail.com' },
        { label: '@mlko01', href: 'https://github.com/mlko01' },
      ],
    },
    {
      heading: t('footer.platform'),
      lead: t('footer.openCore'),
      links: [
        { label: t('footer.releasing'), href: `${REPO_URL}/blob/main/RELEASING.md` },
        { label: t('footer.changelog'), href: `${REPO_URL}/blob/main/CHANGELOG.md` },
        { label: t('footer.contributing'), href: `${REPO_URL}/blob/main/CONTRIBUTING.md` },
      ],
    },
  ]

  return (
    <footer className="border-t border-lyra-borderSoft">
      <div className="mx-auto max-w-7xl px-6 py-12 sm:px-8">
        <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <div className="flex items-center gap-2">
              <LogoMark className="h-6 w-6" />
              <span className="font-display text-sm font-medium text-lyra-goldBright">Lyrahcle</span>
            </div>
            <p className="mt-3 max-w-xs text-sm italic text-lyra-mutedDim">{t('footer.tagline')}</p>
          </div>

          {linkGroups.map((group) => (
            <FooterColumn key={group.heading} {...group} />
          ))}
        </div>

        <div className="mt-10 flex flex-col items-center justify-between gap-2 border-t border-lyra-borderSoft pt-6 sm:flex-row">
          <p className="font-mono text-[11px] text-lyra-mutedDim">
            © {year} Lyrahcle. {t('footer.license')} {t('footer.designedBy')}
          </p>
          <p className="font-mono text-[11px] tracking-[0.2em] text-lyra-mutedDim">
            v{pkg.version}
          </p>
        </div>
      </div>
    </footer>
  )
}
