import { useMemo, useState } from 'react'
import { createOrganization, grantPlatformAdministrator, revokePlatformAdministrator, updateOrganization } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import TableScroll from './TableScroll.jsx'

export default function PlatformOrganizationsPanel({ organizations = [], platformUsers = [], onChanged, onPlatformUsersChanged }) {
  const { t } = useI18n()
  const [slug, setSlug] = useState('')
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const [platformBusyId, setPlatformBusyId] = useState(null)
  const [editingId, setEditingId] = useState(null)
  const [editSlug, setEditSlug] = useState('')
  const [editName, setEditName] = useState('')
  const [editBusy, setEditBusy] = useState(false)

  const organizationById = useMemo(
    () => new Map(organizations.map((organization) => [organization.id, organization])),
    [organizations],
  )

  async function submit(event) {
    event.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await createOrganization(slug.trim().toLowerCase(), name.trim())
      setSlug('')
      setName('')
      await onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  function beginEdit(organization) {
    setError(null)
    setEditingId(organization.id)
    setEditSlug(organization.slug)
    setEditName(organization.name)
  }

  function cancelEdit() {
    setEditingId(null)
    setEditSlug('')
    setEditName('')
  }

  async function saveEdit(event) {
    event.preventDefault()
    setError(null)
    setEditBusy(true)
    try {
      await updateOrganization(editingId, editSlug.trim().toLowerCase(), editName.trim())
      cancelEdit()
      await onChanged()
    } catch (err) {
      setError(err.message)
    } finally {
      setEditBusy(false)
    }
  }

  async function togglePlatformAdmin(user) {
    setError(null)
    setPlatformBusyId(user.id)
    try {
      if (user.platform_admin) await revokePlatformAdministrator(user.id)
      else await grantPlatformAdministrator(user.id)
      await onPlatformUsersChanged?.()
    } catch (err) {
      setError(err.message)
    } finally {
      setPlatformBusyId(null)
    }
  }

  return (
    <div className="grid gap-6">
      <form onSubmit={submit} className="grid gap-3 rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4 md:grid-cols-[1fr_1.5fr_auto] md:items-end">
        <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
          {t('platform.slug')}
          <input
            required
            pattern="[a-z0-9][a-z0-9-]{1,62}"
            value={slug}
            onChange={(event) => setSlug(event.target.value)}
            placeholder="mission-control"
            className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm normal-case text-lyra-text"
          />
        </label>
        <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
          {t('platform.name')}
          <input
            required
            maxLength={120}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={t('platform.namePlaceholder')}
            className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm normal-case text-lyra-text"
          />
        </label>
        <button disabled={busy} className="rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">
          {busy ? t('common.saving') : t('platform.create')}
        </button>
      </form>

      {error && <p role="alert" className="text-sm text-lyra-bad">{error}</p>}

      {!organizations.length ? (
        <p className="text-sm text-lyra-muted">{t('platform.empty')}</p>
      ) : (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-lyra-borderSoft">
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('platform.name')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('platform.slug')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('platform.status')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">ID</th>
                <th className="pb-2" />
              </tr>
            </thead>
            <tbody>
              {organizations.map((organization) => (
                <tr key={organization.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-3 pr-4 text-lyra-text">{organization.name}</td>
                  <td className="py-3 pr-4 font-mono text-xs text-lyra-goldBright">{organization.slug}</td>
                  <td className="py-3 pr-4">
                    <span className={`rounded border px-2 py-1 font-mono text-[10px] uppercase ${organization.enabled === false ? 'border-lyra-bad/50 text-lyra-bad' : 'border-lyra-ok/50 text-lyra-ok'}`}>
                      {organization.enabled === false ? t('platform.disabled') : t('platform.enabled')}
                    </span>
                  </td>
                  <td className="py-3 pr-4 font-mono text-[11px] text-lyra-mutedDim">{organization.id}</td>
                  <td className="py-3 text-right">
                    <button
                      type="button"
                      onClick={() => beginEdit(organization)}
                      className="rounded border border-lyra-border px-3 py-1.5 font-mono text-xs text-lyra-goldBright"
                    >
                      {t('platform.edit')}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}

      {editingId && (
        <form onSubmit={saveEdit} aria-label={t('platform.editTitle')} className="grid gap-3 rounded border border-lyra-borderSoft bg-lyra-bgSoft/30 p-4 md:grid-cols-[1fr_1.5fr_auto_auto] md:items-end">
          <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            {t('platform.editSlug')}
            <input
              required
              pattern="[a-z0-9][a-z0-9-]{1,62}"
              value={editSlug}
              onChange={(event) => setEditSlug(event.target.value)}
              className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-sm normal-case text-lyra-text"
            />
          </label>
          <label className="grid gap-1 text-[11px] uppercase tracking-wider text-lyra-mutedDim">
            {t('platform.editName')}
            <input
              required
              maxLength={120}
              value={editName}
              onChange={(event) => setEditName(event.target.value)}
              className="rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 text-sm normal-case text-lyra-text"
            />
          </label>
          <button disabled={editBusy} className="rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg disabled:opacity-50">
            {editBusy ? t('common.saving') : t('platform.saveChanges')}
          </button>
          <button type="button" disabled={editBusy} onClick={cancelEdit} className="rounded border border-lyra-border px-4 py-2 text-sm text-lyra-muted disabled:opacity-50">
            {t('common.cancel')}
          </button>
        </form>
      )}

      <div className="grid gap-3">
        <div>
          <h3 className="font-mono text-xs uppercase tracking-wider text-lyra-gold">{t('platform.admins')}</h3>
          <p className="mt-1 text-xs text-lyra-muted">{t('platform.adminsHint')}</p>
        </div>
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead><tr className="border-b border-lyra-borderSoft">
              <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('platform.user')}</th>
              <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('platform.organization')}</th>
              <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('platform.access')}</th>
              <th className="pb-2" />
            </tr></thead>
            <tbody>
              {platformUsers.map((user) => {
                const organization = organizationById.get(user.organization_id)
                return (
                  <tr key={user.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                    <td className="py-3 pr-4"><span className="text-lyra-text">{user.username || user.email}</span><p className="text-xs text-lyra-muted">{user.email}</p></td>
                    <td className="py-3 pr-4">
                      <span className="text-xs text-lyra-text">{organization?.name || t('platform.unknownOrganization')}</span>
                      <p className="font-mono text-[10px] text-lyra-mutedDim">{organization?.slug || user.organization_id}</p>
                    </td>
                    <td className="py-3 pr-4 font-mono text-xs">{user.platform_admin ? t('platform.platformAdmin') : t('platform.tenantUser')}</td>
                    <td className="py-3 text-right">
                      <button
                        type="button"
                        disabled={platformBusyId === user.id || !user.enabled}
                        onClick={() => togglePlatformAdmin(user)}
                        className="rounded border border-lyra-border px-3 py-1.5 font-mono text-xs text-lyra-goldBright disabled:opacity-50"
                      >
                        {user.platform_admin ? t('platform.revokeAdmin') : t('platform.grantAdmin')}
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </TableScroll>
      </div>
    </div>
  )
}
