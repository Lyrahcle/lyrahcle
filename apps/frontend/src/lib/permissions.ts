import { fetchEffectivePermissions } from './api'

export type Permission = string

export interface EffectivePermissionsResponse {
  permissions: Permission[]
  [key: string]: unknown
}

let permissions: Set<Permission> = new Set()
let loaded = false

export function setEffectivePermissions(values: Permission[] = []): void {
  permissions = new Set(Array.isArray(values) ? values : [])
  loaded = true
}

export function clearEffectivePermissions(): void {
  permissions = new Set()
  loaded = false
}

export function permissionsLoaded(): boolean {
  return loaded
}

export function hasPermission(permission: Permission): boolean {
  return permissions.has(permission)
}

export function effectivePermissions(): Permission[] {
  return [...permissions]
}

export async function loadEffectivePermissions(): Promise<EffectivePermissionsResponse> {
  try {
    const data = (await fetchEffectivePermissions()) as EffectivePermissionsResponse
    setEffectivePermissions(data.permissions)
    return data
  } catch (error) {
    clearEffectivePermissions()
    throw error
  }
}
