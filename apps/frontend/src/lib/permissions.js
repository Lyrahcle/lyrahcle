import { fetchEffectivePermissions } from './api.js'

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
  try {
    const data = await fetchEffectivePermissions()
    setEffectivePermissions(data.permissions)
    return data
  } catch (error) {
    clearEffectivePermissions()
    throw error
  }
}
