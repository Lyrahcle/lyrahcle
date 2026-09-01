import { useEffect, useMemo, useState } from 'react'
import {
  assignUserRBACRole,
  createRBACRole,
  fetchUserRBACRoles,
  removeUserRBACRole,
} from '../lib/api.js'
import { hasPermission } from '../lib/permissions.js'
import { useI18n } from '../lib/I18nContext.jsx'
import TableScroll from './TableScroll.jsx'

export default function RBACPanel({ permissions, roles, users, onChanged }) {
  const { t } = useI18n()
  const canManage = hasPermission('rbac.manage')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selectedPermissions, setSelectedPermissions] = useState([])
  const [selectedUser, setSelectedUser] = useState('')
  const [assignedRoles, setAssignedRoles] = useState([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  const groupedPermissions = useMemo(() => {
    const groups = new Map()
    for (const permission of permissions ?? []) {
      const domain = permission.key.split('.')[0]
      if (!groups.has(domain)) groups.set(domain, [])
      groups.get(domain).push(permission)
    }
    return [...groups.entries()]
  }, [permissions])

  useEffect(() => {
    if (!selectedUser) {
      setAssignedRoles([])
      return
    }
    let cancelled = false
    setError(null)
    fetchUserRBACRoles(selectedUser)
      .then((items) => {
        if (!cancelled) setAssignedRoles(items ?? [])
      })
      .catch((err) => {
        if (!cancelled) setError(err.message)
      })
    return () => { cancelled = true }
  }, [selectedUser, roles])

  async function createRole(event) {
    event.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await createRBACRole(name.trim(), description.trim(), selectedPermissions)
      setName('')
      setDescription('')
      setSelectedPermissions([])
      await onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  function togglePermission(key) {
    setSelectedPermissions((current) => current.includes(key) ? current.filter((item) => item !== key) : [...current, key])
  }

  async function toggleRole(role) {
    if (!selectedUser || !canManage) return
    setBusy(true)
    setError(null)
    try {
      const assigned = assignedRoles.some((item) => item.id === role.id)
      if (assigned) await removeUserRBACRole(selectedUser, role.id)
      else await assignUserRBACRole(selectedUser, role.id)
      setAssignedRoles(await fetchUserRBACRoles(selectedUser))
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="grid gap-6">
      {error && <p role="alert" className="text-sm text-lyra-bad">{error}</p>}

      {canManage && (
        <form onSubmit={createRole} className="grid gap-4 rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4">
          <div className="grid gap-3 md:grid-cols-2">
            <label className="grid gap-1 text-[11px] tracking-wider text-lyra-mutedDim">
              {t('rbac.roleName')}
              <input required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm normal-case text-lyra-text" />
            </label>
            <label className="grid gap-1 text-[11px] tracking-wider text-lyra-mutedDim">
              {t('rbac.description')}
              <input maxLength={240} value={description} onChange={(e) => setDescription(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm normal-case text-lyra-text" />
            </label>
          </div>

          <div>
            <p className="mb-2 font-mono text-[11px] tracking-wider text-lyra-mutedDim">{t('rbac.capabilities')}</p>
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {groupedPermissions.map(([domain, items]) => (
                <fieldset key={domain} className="rounded border border-lyra-borderSoft p-3">
                  <legend className="px-1 font-mono text-xs text-lyra-goldBright">{domain}</legend>
                  <div className="grid gap-2">
                    {items.map((permission) => (
                      <label key={permission.key} className="flex gap-2 text-xs text-lyra-muted">
                        <input type="checkbox" checked={selectedPermissions.includes(permission.key)} onChange={() => togglePermission(permission.key)} />
                        <span><span className="font-mono text-lyra-text">{permission.key}</span>{permission.description ? ` — ${permission.description}` : ''}</span>
                      </label>
                    ))}
                  </div>
                </fieldset>
              ))}
            </div>
          </div>

          <button disabled={busy} className="justify-self-start rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">
            {busy ? t('common.saving') : t('rbac.createRole')}
          </button>
        </form>
      )}

      <div>
        <h3 className="mb-3 font-mono text-xs tracking-wider text-lyra-gold">{t('rbac.roles')}</h3>
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead><tr className="border-b border-lyra-borderSoft">
              <th className="pb-2 text-[11px] text-lyra-mutedDim">{t('rbac.role')}</th>
              <th className="pb-2 text-[11px] text-lyra-mutedDim">{t('rbac.type')}</th>
              <th className="pb-2 text-[11px] text-lyra-mutedDim">{t('rbac.capabilities')}</th>
            </tr></thead>
            <tbody>
              {(roles ?? []).map((role) => (
                <tr key={role.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-3 pr-4"><span className="text-lyra-text">{role.name}</span>{role.description && <p className="mt-1 text-xs text-lyra-muted">{role.description}</p>}</td>
                  <td className="py-3 pr-4 font-mono text-xs text-lyra-mutedDim">{role.system ? t('rbac.system') : t('rbac.custom')}</td>
                  <td className="py-3 font-mono text-[11px] text-lyra-muted">{(role.permissions ?? []).map((permission) => permission.key).join(', ') || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      </div>

      {users?.length > 0 && (
        <div className="grid gap-3">
          <label className="grid max-w-xl gap-1 text-[11px] tracking-wider text-lyra-mutedDim">
            {t('rbac.user')}
            <select value={selectedUser} onChange={(e) => setSelectedUser(e.target.value)} className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm normal-case text-lyra-text">
              <option value="">{t('rbac.selectUser')}</option>
              {users.map((user) => <option key={user.id} value={user.id}>{user.username || user.email}</option>)}
            </select>
          </label>

          {selectedUser && (
            <div className="flex flex-wrap gap-2">
              {(roles ?? []).map((role) => {
                const assigned = assignedRoles.some((item) => item.id === role.id)
                return (
                  <button
                    key={role.id}
                    type="button"
                    disabled={!canManage || busy}
                    aria-pressed={assigned}
                    onClick={() => toggleRole(role)}
                    className={`rounded border px-3 py-1.5 font-mono text-xs ${assigned ? 'border-lyra-good/50 text-lyra-good' : 'border-lyra-border text-lyra-muted'} disabled:opacity-50`}
                  >
                    {role.name}
                  </button>
                )
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
