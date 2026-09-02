export interface RBACPermission {
  key: string
  description?: string
  [key: string]: unknown
}

export interface RBACRole {
  id: string
  name: string
  description?: string
  system?: boolean
  permissions?: RBACPermission[]
  [key: string]: unknown
}

export interface RBACUser {
  id: string
  email?: string
  username?: string
  [key: string]: unknown
}
