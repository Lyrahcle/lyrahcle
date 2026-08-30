import { useState } from 'react'
import { createOrganization, grantPlatformAdministrator, revokePlatformAdministrator } from '../lib/api.js'
import { useI18n } from '../lib/I18nContext.jsx'
import TableScroll from './TableScroll.jsx'

export default function PlatformOrganizationsPanel({ organizations, platformUsers = [], onChanged, onPlatformUsersChanged }) {
  const { t } = useI18n()
  const [slug, setSlug] = useState('')
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const [platformBusyId, setPlatformBusyId] = useState(null)

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


  async function togglePlatformAdmin(user) {
    setError(null)
    setPlatformBusyId(user.id)
    try {
      if (user.platform_admin) await revokePlatformAdministrator(user.id)
      else await grantPlatformAdministrator(user.id)
      await onPlatformUsersChanged()
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

      {!organizations?.length ? (
        <p className="text-sm text-lyra-muted">{t('platform.empty')}</p>
      ) : (
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-lyra-borderSoft">
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('platform.name')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">{t('platform.slug')}</th>
                <th className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">ID</th>
              </tr>
            </thead>
            <tbody>
              {organizations.map((organization) => (
                <tr key={organization.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-3 pr-4 text-lyra-text">{organization.name}</td>
                  <td className="py-3 pr-4 font-mono text-xs text-lyra-goldBright">{organization.slug}</td>
                  <td className="py-3 font-mono text-[11px] text-lyra-mutedDim">{organization.id}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
      )}

      <div className="grid gap-3">
        <h3 className="font-mono text-xs uppercase tracking-wider text-lyra-gold">{t('platform.admins')}</h3>
        <TableScroll>
          <table className="w-full text-left text-sm">
            <thead><tr className="border-b border-lyra-borderSoft">
              <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('platform.user')}</th>
              <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('platform.organization')}</th>
              <th className="pb-2 text-[11px] uppercase text-lyra-mutedDim">{t('platform.access')}</th>
              <th className="pb-2" />
            </tr></thead>
            <tbody>
              {platformUsers.map((user) => (
                <tr key={user.id} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="py-3 pr-4"><span className="text-lyra-text">{user.username || user.email}</span><p className="text-xs text-lyra-muted">{user.email}</p></td>
                  <td className="py-3 pr-4 font-mono text-[11px] text-lyra-mutedDim">{user.organization_id}</td>
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
              ))}
            </tbody>
          </table>
        </TableScroll>
      </div>
    </div>
  )
}
