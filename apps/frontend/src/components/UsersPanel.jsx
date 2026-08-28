import { useState } from 'react'
import { updateUserRole } from '../lib/api.js'
import { decodeToken, getToken } from '../lib/auth.js'
import TableScroll from './TableScroll.jsx'

const ROLES = ['viewer', 'operator', 'admin']

export default function UsersPanel({ users, onChanged }) {
  const [error, setError] = useState(null)
  const [savingId, setSavingId] = useState(null)

  const currentUserId = decodeToken(getToken())?.user_id

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

  return (
    <div>
      {error && <p className="mb-4 text-sm text-lyra-bad">{error}</p>}

      {!users.length ? (
        <p className="text-sm text-lyra-muted">No users found.</p>
      ) : (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-lyra-borderSoft">
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                  Email
                </th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                  Role
                </th>
                <th className="pb-2" />
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-2.5 font-mono text-xs text-lyra-goldBright">
                    {u.email}
                    {u.id === currentUserId && (
                      <span className="ml-2 rounded-full border border-lyra-border px-2 py-0.5 text-[10px] uppercase tracking-wider text-lyra-mutedDim">
                        you
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
                  <td className="py-2.5 text-right font-mono text-[11px] text-lyra-mutedDim">
                    {savingId === u.id ? 'Saving…' : ''}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}
    </div>
  )
}
