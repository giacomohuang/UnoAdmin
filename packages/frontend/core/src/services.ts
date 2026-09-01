import type { Permission, Role, Tenant, User } from './entities'
import type { UserId } from './identifiers'
import type { ListQuery, PageResult } from './query'

/** Service collection required by the reusable base administration package. */
export interface BaseServices {
  users: UserService
  roles: RoleService
  permissions: PermissionService
  tenants: TenantService
}

/** User service contract used by the base administration screens. */
export interface UserService {
  list(query: ListQuery): Promise<PageResult<User>>
  updateStatus(id: UserId, status: User['status']): Promise<void>
}

/** Role service contract used by the base administration screens. */
export interface RoleService {
  list(query: ListQuery): Promise<PageResult<Role>>
}

/** Permission service contract used by the base administration screens. */
export interface PermissionService {
  list(): Promise<Permission[]>
}

/** Tenant service contract used by the base administration screens. */
export interface TenantService {
  list(query: ListQuery): Promise<PageResult<Tenant>>
}
