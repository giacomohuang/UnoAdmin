import type { TenantId } from './identifiers'
import type { Tenant, User } from './entities'

/** Authentication state restored by an application-specific auth adapter. */
export interface AuthSession {
  user: User
  permissions: string[]
  tenants: Tenant[]
  currentTenantId: TenantId | null
}

/** Authentication adapter used to restore and clear the current session. */
export interface AuthPort {
  restore(): Promise<AuthSession | null>
  signOut(): Promise<void>
}
