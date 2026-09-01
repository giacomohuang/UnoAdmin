/** A branded string keeps identifiers from different resources type-safe. */
export type Brand<Value, Name extends string> = Value & { readonly __brand: Name }

/** Stable identifier for users. */
export type UserId = Brand<string, 'UserId'>

/** Stable identifier for roles. */
export type RoleId = Brand<string, 'RoleId'>

/** Stable identifier for permissions. */
export type PermissionId = Brand<string, 'PermissionId'>

/** Stable identifier for tenants. */
export type TenantId = Brand<string, 'TenantId'>
