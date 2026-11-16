import type { AuthPort, SignInCredentials } from './auth'
import type { BaseServices } from './services'
import type { Tenant, User } from './types/entities'
import type { TenantId } from './types/identifiers'

/** A navigation item is declared by a module and filtered by permissions. */
export interface NavigationItem {
  id: string
  label: string
  to?: string
  permission?: string
  children?: NavigationItem[]
}

/** Permission metadata published by a module. */
export interface PermissionDefinition {
  code: string
  label: string
  description?: string
}

/** The package-level contract consumed by the application composition root. */
export interface AdminModule {
  id: string
  routeNamespace: string
  navigation: NavigationItem[]
  permissions: PermissionDefinition[]
  install?: (context: AdminModuleContext) => void | (() => void)
}

/** Context passed to an optional module installer hook. */
export interface AdminModuleContext {
  services: unknown
  permissions: ReadonlySet<string>
}

/** All services are supplied by the application so packages remain backend-neutral. */
export interface AppServices {
  base: BaseServices
}

/** Optional brand values consumed by the base layout. */
export interface AdminBranding {
  productName: string
  productMark: string
}

/** Mutable authentication and tenant state owned by the application shell. */
export interface AdminSessionState {
  ready: boolean
  user: User | null
  permissions: string[]
  tenants: Tenant[]
  currentTenant: Tenant | null
}

/** Recursively marks session state as read-only without depending on Vue types. */
export type DeepReadonly<T> = T extends string | number | boolean | bigint | symbol | null | undefined
  ? T
  : T extends (...args: never[]) => unknown
    ? T
    : T extends readonly (infer Item)[]
      ? readonly DeepReadonly<Item>[]
      : T extends object
        ? { readonly [Key in keyof T]: DeepReadonly<T[Key]> }
        : T

/** Application session methods consumed by shell and business composables. */
export interface AdminSession {
  readonly state: DeepReadonly<AdminSessionState>
  /**
   * Establish a session through the configured authentication adapter.
   * @returns Promise resolved after the session state is updated.
   */
  signIn(credentials?: SignInCredentials): Promise<void>
  initialize(): Promise<void>
  can(permission?: string): boolean
  switchTenant(id: TenantId): Promise<void>
  signOut(): Promise<void>
}

/** Inputs supplied by the concrete application that owns session creation. */
export interface CreateAdminSessionOptions {
  auth: AuthPort
}

/** Shared injection tokens used by concrete application and business composables. */
export const ADMIN_SESSION_KEY = Symbol.for('unoadmin.admin-session')
export const ADMIN_BRANDING_KEY = Symbol.for('unoadmin.admin-branding')
export const ADMIN_MODULES_KEY = Symbol.for('unoadmin.admin-modules')
export const ADMIN_SERVICES_KEY = Symbol.for('unoadmin.admin-services')
