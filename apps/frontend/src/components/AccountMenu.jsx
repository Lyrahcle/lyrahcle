import { useEffect, useRef, useState } from 'react'
import ChangePasswordModal from './ChangePasswordModal.jsx'
import { getRole } from '../lib/auth.js'
import { useI18n } from '../lib/I18nContext.jsx'

const ROLE_DOT_COLOR = {
  admin: 'bg-lyra-gold',
  operator: 'bg-lyra-good',
  viewer: 'bg-lyra-muted',
}

export default function AccountMenu({ onLogout }) {
  const { t } = useI18n()
  const [menuOpen, setMenuOpen] = useState(false)
  const [changingPassword, setChangingPassword] = useState(false)
  const menuRef = useRef(null)
  const role = getRole()

  useEffect(() => {
    if (!menuOpen) return
    function handleClickOutside(e) {
      if (menuRef.current && !menuRef.current.contains(e.target)) setMenuOpen(false)
    }
    function handleEscape(e) {
      if (e.key === 'Escape') setMenuOpen(false)
    }
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [menuOpen])

  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={() => setMenuOpen((v) => !v)}
        aria-haspopup="true"
        aria-expanded={menuOpen}
        className="flex items-center gap-2 rounded border border-lyra-border px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
      >
        {role && <span className={`h-1.5 w-1.5 rounded-full ${ROLE_DOT_COLOR[role] ?? 'bg-lyra-gold'}`} />}
        {role ?? t('header.account')}
        <span className="text-lyra-mutedDim">▾</span>
      </button>

      {menuOpen && (
        <div className="absolute right-0 top-full mt-2 w-48 overflow-hidden rounded border border-lyra-border bg-lyra-panel shadow-lg">
          <button
            onClick={() => {
              setChangingPassword(true)
              setMenuOpen(false)
            }}
            className="block w-full px-4 py-2.5 text-left font-mono text-[11px] uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:bg-lyra-bgSoft hover:text-lyra-goldBright"
          >
            {t('header.changePassword')}
          </button>
          <button
            onClick={() => {
              setMenuOpen(false)
              onLogout()
            }}
            className="block w-full border-t border-lyra-borderSoft px-4 py-2.5 text-left font-mono text-[11px] uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:bg-lyra-bgSoft hover:text-lyra-bad"
          >
            {t('header.signOut')}
          </button>
        </div>
      )}

      {changingPassword && <ChangePasswordModal onClose={() => setChangingPassword(false)} />}
    </div>
  )
}
