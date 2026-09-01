import { useMemo, useState } from 'react'
import { Search } from 'lucide-react'
import { createUser, deleteUser, updateUserRole, updateUserStatus } from '../lib/api.js'
import { decodeToken, getToken } from '../lib/auth.js'
import { useI18n } from '../lib/I18nContext.jsx'
import ConfirmDialog from './ConfirmDialog.jsx'
import TableScroll from './TableScroll.jsx'

const ROLES = ['viewer', 'auditor', 'analyst', 'operator', 'admin']

function NewUserForm({ onCreated }) {
  const { t } = useI18n()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('viewer')
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await createUser(email, password, role)
      setEmail('')
      setPassword('')
      setRole('viewer')
      onCreated()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="mb-6 flex flex-wrap items-end gap-3 rounded border border-lyra-border bg-lyra-bgSoft/50 p-4"
    >
      <div className="flex-1 min-w-[10rem]">
        <label htmlFor="new-user-email" className="mb-1.5 block text-[11px] tracking-wider text-lyra-mutedDim">
          {t('common.email')}
        </label>
        <input
          id="new-user-email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="w-full rounded border border-lyra-border bg-lyra-bg px-2.5 py-1.5 text-sm text-lyra-text outline-none focus:border-lyra-gold"
          required
        />
      </div>
      <div className="flex-1 min-w-[10rem]">
        <label htmlFor="new-user-password" className="mb-1.5 block text-[11px] tracking-wider text-lyra-mutedDim">
          {t('common.password')}
        </label>
        <input
          id="new-user-password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full rounded border border-lyra-border bg-lyra-bg px-2.5 py-1.5 text-sm text-lyra-text outline-none focus:border-lyra-gold"
          required
          minLength={12}
          maxLength={72}
        />
      </div>
      <div>
        <label htmlFor="new-user-role" className="mb-1.5 block text-[11px] tracking-wider text-lyra-mutedDim">
          {t('users.role')}
        </label>
        <select
          id="new-user-role"
          value={role}
          onChange={(e) => setRole(e.target.value)}
          className="rounded border border-lyra-border bg-lyra-bg px-2.5 py-1.5 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
        >
          {ROLES.map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </select>
      </div>
      <button
        type="submit"
        disabled={loading}
        className="rounded bg-lyra-gold px-4 py-1.5 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
      >
        {loading ? t('users.creating') : t('users.createUser')}
      </button>
      {error && (
        <p role="alert" className="w-full text-sm text-lyra-bad">
          {error}
        </p>
      )}
    </form>
  )
}

export default function UsersPanel({ users, onChanged }) {
  const { t } = useI18n()
  const [error, setError] = useState(null)
  const [savingId, setSavingId] = useState(null)
  const [query, setQuery] = useState('')
  const [pendingDeleteId, setPendingDeleteId] = useState(null)

  const currentUserId = decodeToken(getToken())?.user_id

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return users
    return users.filter((u) => `${u.email} ${u.username ?? ''} ${u.role}`.toLowerCase().includes(q))
  }, [users, query])

  async function handleRoleChange(userId, role) {
    setError(null)
    setSavingId(userId)
    try {
      await updateUserRole(userId, role)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSavingId(null)
    }
  }

  async function handleStatusChange(userId, enabled) {
    setError(null)
    setSavingId(userId)
    try {
      await updateUserStatus(userId, enabled)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSavingId(null)
    }
  }

  async function confirmDelete() {
    const userId = pendingDeleteId
    if (!userId) return
    setError(null)
    setSavingId(userId)
    try {
      await deleteUser(userId)
      setPendingDeleteId(null)
      onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setSavingId(null)
    }
  }

  if (!users.length) {
    return (
      <div>
        <NewUserForm onCreated={onChanged} />
        <p className="text-sm text-lyra-muted">{t('users.noUsers')}</p>
      </div>
    )
  }

  return (
    <div>
      <NewUserForm onCreated={onChanged} />

      <div className="mb-4 flex items-center justify-between gap-3">
        <label className="group flex flex-1 max-w-xs items-center gap-2 rounded border border-lyra-border bg-lyra-bgSoft px-3 py-1.5 transition-colors focus-within:border-lyra-gold/50">
          <Search className="h-3.5 w-3.5 text-lyra-mutedDim group-focus-within:text-lyra-gold" strokeWidth={1.75} />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            type="text"
            placeholder={t('users.searchPlaceholder')}
            className="w-full bg-transparent text-sm text-lyra-text placeholder:text-lyra-mutedDim focus:outline-none"
          />
        </label>
        <p className="font-mono text-[11px] tracking-wider text-lyra-mutedDim">
          {t('users.count', { n: filtered.length, total: users.length })}
        </p>
      </div>

      {error && (
            <p role="alert" className="mb-4 text-sm text-lyra-bad">
              {error}
            </p>
          )}

      {!filtered.length ? (
        <p className="text-sm text-lyra-muted">{t('users.noMatches')}</p>
      ) : (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-lyra-borderSoft">
                <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                  {t('users.email')}
                </th>
                <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                  {t('users.role')}
                </th>
                <th scope="col" className="pb-2 text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                  {t('users.status')}
                </th>
                <th scope="col" className="pb-2 text-right text-[11px] font-normal tracking-wider text-lyra-mutedDim">
                  {t('users.actions')}
                </th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((u) => (
                <tr key={u.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-2.5 font-mono text-xs text-lyra-goldBright">
                    {u.email}
                    {u.id === currentUserId && (
                      <span className="ml-2 rounded-full border border-lyra-border px-2 py-0.5 text-[10px] tracking-wider text-lyra-mutedDim">
                        {t('common.you')}
                      </span>
                    )}
                  </td>
                  <td className="py-2.5">
                    <select
                      value={u.role}
                      disabled={savingId === u.id}
                      onChange={(e) => handleRoleChange(u.id, e.target.value)}
                      className="rounded border border-lyra-border bg-lyra-bgSoft px-2 py-1 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold disabled:opacity-50"
                    >
                      {ROLES.map((role) => (
                        <option key={role} value={role}>
                          {role}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="py-2.5">
                    <button
                      type="button"
                      disabled={savingId === u.id || u.id === currentUserId}
                      onClick={() => handleStatusChange(u.id, !u.enabled)}
                      className="rounded border border-lyra-border px-2 py-1 font-mono text-[11px] text-lyra-muted outline-none hover:text-lyra-goldBright disabled:opacity-40"
                    >
                      {u.enabled ? t('users.active') : t('users.disabled')}
                    </button>
                  </td>
                  <td className="py-2.5 text-right">
                    {u.id !== currentUserId && (
                      <button
                        type="button"
                        disabled={savingId === u.id}
                        onClick={() => setPendingDeleteId(u.id)}
                        className="rounded border border-lyra-bad/40 px-2 py-1 font-mono text-[11px] text-lyra-bad outline-none hover:bg-lyra-bad/10 disabled:opacity-40"
                      >
                        {t('users.delete')}
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
      {pendingDeleteId && (
        <ConfirmDialog
          message={t('users.deleteConfirm')}
          busy={savingId === pendingDeleteId}
          onConfirm={confirmDelete}
          onCancel={() => setPendingDeleteId(null)}
        />
      )}
    </div>
  )
}
