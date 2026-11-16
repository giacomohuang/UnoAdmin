import type { Tenant, User } from './types/entities'
import type { TenantId } from './types/identifiers'

/** Authentication state restored by an application-specific auth adapter. */
export interface AuthSession {
  user: User
  permissions: string[]
  tenants: Tenant[]
  currentTenantId: TenantId | null
}

/** Credentials accepted by a real authentication backend. */
export interface SignInCredentials {
  username: string
  password: string
}

/**
 * Authentication adapter used to enter, restore, and clear the current session.
 *
 * Credentials are optional so local adapters can still sign in without input;
 * HTTP adapters require them and surface an error when missing.
 */
export interface AuthPort {
  signIn(credentials?: SignInCredentials): Promise<AuthSession>
  restore(): Promise<AuthSession | null>
  signOut(): Promise<void>
  /** Persist a tenant switch in the backend when the adapter supports it. */
  switchTenant?(id: TenantId): Promise<void>
}
