import type { PermissionId, RoleId, TenantId, UserId } from './identifiers'

/** A tenant visible to the current authenticated session. */
export interface Tenant {
  id: TenantId
  name: string
  code: string
  status: 'active' | 'suspended'
}

/** The compact user model used by the base administration screens. */
export interface User {
  id: UserId
  username: string
  displayName: string
  email: string
  status: 'active' | 'disabled'
  roleNames: string[]
}

/** A role and its effective permission codes. */
export interface Role {
  id: RoleId
  name: string
  description: string
  permissionCodes: string[]
  userCount: number
}

/** A permission node that can be rendered as a tree. */
export interface Permission {
  id: PermissionId
  name: string
  code: string
  type: 'menu' | 'page' | 'action'
  children?: Permission[]
}
