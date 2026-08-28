import { useState } from 'react'
import { changePassword } from '../lib/api.js'

export default function ChangePasswordModal({ onClose }) {
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState(null)
  const [success, setSuccess] = useState(false)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match')
      return
    }

    setLoading(true)
    try {
      await changePassword(currentPassword, newPassword)
      setSuccess(true)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-6">
      <div className="w-full max-w-sm rounded-lg border border-lyra-border bg-lyra-panel p-6">
        <div className="mb-5 flex items-center justify-between">
          <h2 className="font-display text-sm font-medium text-lyra-goldBright">Change password</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="rounded text-lyra-muted outline-none transition-colors hover:text-lyra-goldBright focus-visible:ring-1 focus-visible:ring-lyra-gold"
          >
            ✕
          </button>
        </div>

        {success ? (
          <div>
            <p className="mb-5 text-sm text-lyra-good">Password updated successfully.</p>
            <button
              type="button"
              onClick={onClose}
              className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright"
            >
              Done
            </button>
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <div className="mb-4">
              <label htmlFor="current-password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                Current password
              </label>
              <input
                id="current-password"
                type="password"
                autoComplete="current-password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
              />
            </div>

            <div className="mb-4">
              <label htmlFor="new-password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                New password
              </label>
              <input
                id="new-password"
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
                minLength={8}
              />
            </div>

            <div className="mb-5">
              <label htmlFor="confirm-new-password" className="mb-1.5 block text-xs uppercase tracking-wider text-lyra-mutedDim">
                Confirm new password
              </label>
              <input
                id="confirm-new-password"
                type="password"
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm text-lyra-text outline-none focus:border-lyra-gold"
                required
                minLength={8}
              />
            </div>

            {error && <p className="mb-4 text-sm text-lyra-bad">{error}</p>}

            <div className="flex gap-3">
              <button
                type="button"
                onClick={onClose}
                className="w-full rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted outline-none transition-colors hover:text-lyra-text focus-visible:ring-1 focus-visible:ring-lyra-gold"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="w-full rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
              >
                {loading ? 'Saving…' : 'Save'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
