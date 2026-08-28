import { useEffect, useRef, useState } from 'react'
import LogoMark from './LogoMark.jsx'
import Clock from './Clock.jsx'
import ChangePasswordModal from './ChangePasswordModal.jsx'
import { getRole } from '../lib/auth.js'

export default function Header({ onLogout, onHome }) {
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
    <header className="sticky top-0 z-40 border-b border-lyra-borderSoft bg-lyra-bg/85 backdrop-blur">
      <div className="mx-auto flex max-w-7xl items-center justify-between gap-3 px-6 py-3.5 sm:px-8">
        <button
          onClick={onHome}
          className="flex items-center gap-2.5 rounded outline-none focus-visible:ring-1 focus-visible:ring-lyra-gold"
        >
          <LogoMark className="h-6 w-6" />
          <span className="font-display text-[15px] font-medium tracking-tight text-lyra-goldBright">
            Lyrahcle
          </span>
          <span className="hidden font-mono text-[10px] uppercase tracking-[0.25em] text-lyra-mutedDim sm:inline">
            Constellation Analytics
          </span>
        </button>

        <div className="flex items-center gap-4">
          <Clock />

          {onLogout && (
            <div className="relative" ref={menuRef}>
              <button
                onClick={() => setMenuOpen((v) => !v)}
                aria-haspopup="true"
                aria-expanded={menuOpen}
                className="flex items-center gap-2 rounded border border-lyra-border px-3 py-1.5 font-mono text-[11px] uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:border-lyra-gold/40 hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
              >
                {role && <span className="h-1.5 w-1.5 rounded-full bg-lyra-gold" />}
                {role ?? 'account'}
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
                    Change password
                  </button>
                  <button
                    onClick={() => {
                      setMenuOpen(false)
                      onLogout()
                    }}
                    className="block w-full border-t border-lyra-borderSoft px-4 py-2.5 text-left font-mono text-[11px] uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:bg-lyra-bgSoft hover:text-lyra-bad"
                  >
                    Sign out
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {changingPassword && <ChangePasswordModal onClose={() => setChangingPassword(false)} />}
    </header>
  )
}
