import { authHeaders } from './auth.js'

let permissions = new Set()
let loaded = false

export function setEffectivePermissions(values = []) {
  permissions = new Set(Array.isArray(values) ? values : [])
  loaded = true
}

export function clearEffectivePermissions() {
  permissions = new Set()
  loaded = false
}

export function permissionsLoaded() {
  return loaded
}

export function hasPermission(permission) {
  return permissions.has(permission)
}

export function effectivePermissions() {
  return [...permissions]
}

export async function loadEffectivePermissions() {
  const res = await fetch('/api/v1/rbac/me', { headers: authHeaders() })
  if (!res.ok) {
    clearEffectivePermissions()
    throw new Error('failed to load effective permissions')
  }
  const data = await res.json()
  setEffectivePermissions(data.permissions)
  return data
}
