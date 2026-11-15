import type { Tenant, User } from './types/entities'
import type { TenantId } from './types/identifiers'

/** Authentication state restored by an application-specific auth adapter. */
export interface AuthSession {
  user: User
  permissions: string[]
  tenants: Tenant[]
  currentTenantId: TenantId | null
}

/** Authentication adapter used to enter, restore, and clear the current session. */
export interface AuthPort {
  signIn(): Promise<AuthSession>
  restore(): Promise<AuthSession | null>
  signOut(): Promise<void>
}
